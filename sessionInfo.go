package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SessionsData структура для выгрузки информации по сессиям
type SessionsData struct {
	Sessions []struct {
		SessionUuid  string `json:"uuid"`
		ClientId     string `json:"client_id"`
		ProductId    string `json:"product_id"`
		CreatedOn    int64  `json:"created_on"`
		FinishedOn   int64  `json:"finished_on"` //or null
		Status       string `json:"status"`
		CreatorIp    string `json:"creator_ip"`
		AbortComment string `json:"abort_comment"` //or null
		Score        string `json:"score"`         //or null
		ScoreReason  string `json:"score_reason"`  //or null
		Comment      string `json:"score_text"`    //or null
		BillingType  string `json:"billing_type"`  // or null
	}
}

func (a *App) sessionInfo(status string) (infoString string) {
	var sumTrial int
	var serverIP, sessionId, ipInfo string
	var hname string
	fileGames := filepath.Join(a.appDir, "trial.txt")

	if status == "Start" { // формируем текст для отправки
		responseString, err := a.getFromURL(UrlSessions, "server_id", a.cfg.serverID)
		if err != nil {
			infoString = hname + "невозможно получить данные с сайта"
			log.Println("[ERROR] Невозможно получить данные с сайта")
		} else {
			var data SessionsData                                // структура SessionsData
			err := json.Unmarshal([]byte(responseString), &data) // декодируем JSON файл
			if err != nil {
				log.Println("[ERROR] SessionsData unmarshal error: ", err)
			}
			sessionId = data.Sessions[0].SessionUuid
			log.Printf("[INFO] Подключение %s, billing: %s\n", data.Sessions[0].CreatorIp, data.Sessions[0].BillingType)
			game, _ := readConfig(data.Sessions[0].ProductId, fileGames)
			sessionOn, _ := dateTimeS(data.Sessions[0].CreatedOn)
			ipInfo = ""

			if a.cfg.OnlineIpInfo {
				ipInfo = data.Sessions[0].CreatorIp + onlineDBip(data.Sessions[0].CreatorIp)
			} else {
				ipInfo = data.Sessions[0].CreatorIp + offlineDBip(data.Sessions[0].CreatorIp)
			}
			var billing string
			billing = data.Sessions[0].BillingType
			if billing != "" && billing != "trial" {
				billing = data.Sessions[0].BillingType
			}
			if a.cfg.TrialON {
				if billing == "trial" {
					sumTrial = getValueByKey(data.Sessions[0].CreatorIp)
					if sumTrial == -1 { // нет записей по этому IP
						createOrUpdateKeyValue(data.Sessions[0].CreatorIp, 0)
						billing = data.Sessions[0].BillingType
					} else if sumTrial >= 0 && sumTrial < 19 { // уже подключался, но не играл в общей сложности 19 минуту
						billing = fmt.Sprintf("TRIAL %dмин", sumTrial)
					} else if sumTrial > 18 { // начал злоупотреблять
						billing = fmt.Sprintf("TRIAL %dмин\nЗлоупотребление Триалом!", sumTrial)

						if a.cfg.TrialBlock {
							text := "Злоупотребление Триалом! Кикаем!"
							var chatMessage string
							if a.cfg.viewHostname {
								chatMessage = fmt.Sprintf("Внимание! Станция %s.\n%s", a.cfg.hostName, text)
							} else {
								chatMessage = fmt.Sprintf("Внимание!\n%s", text)
							}
							err := a.tg.SendMessage(chatMessage) // отправка сообщения
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
							}
							log.Printf("[INFO] Заблокировано соединение: %s. Trial %d", data.Sessions[0].CreatorIp, sumTrial)
							time.Sleep(10 * time.Second)
							err = runCommand("taskkill", "/IM", "ese.exe", "/F") // закрываем стример сервиса
							if err != nil {
								log.Println("[ERROR] Ошибка выполнения команды:", err)
								return
							}
						}
					}
				}
			}
			localAddr, nameInterface := getInterface()
			serverIP = "\n" + nameInterface + " - " + localAddr
			game = fmt.Sprintf("<b><i> %s </i></b>", game)
			infoHTML := hname + game + "\n" + ipInfo + "\n" + sessionOn + " - " + billing + serverIP
			infoString = "<b>🟢</b>" + infoHTML

		}
	} else if status == "Stop" { // высчитываем продолжительность сессии и формируем текст для отправки
		var minute int
		var sessionDur string
		var stopTime, startTime time.Time
		for i := 0; i < 12; i++ {

			responseString, err := a.getFromURL(UrlSessions, "uuid", sessionId)
			if err != nil {
				log.Println("[ERROR] Stop. Невозможно получить данные с сайта")
			} else {
				var data SessionsData
				err := json.Unmarshal([]byte(responseString), &data) // декодируем JSON файл
				if err != nil {
					log.Println("[ERROR] SessionsData unmarshal error: ", err)
				}
				test := data.Sessions[0].FinishedOn
				if test == 0 {
					time.Sleep(5 * time.Second)
					text := fmt.Sprintf("stopTime = %d", data.Sessions[0].FinishedOn)
					sessionDur = text
				} else {
					_, stopTime = dateTimeS(data.Sessions[0].FinishedOn)
					_, startTime = dateTimeS(data.Sessions[0].CreatedOn)
					sessionDur, minute = dur(stopTime, startTime)
					i = 12
				}
			}
		}

		responseString, err := a.getFromURL(UrlSessions, "uuid", sessionId)
		if err != nil {
			infoString = hname + "невозможно получить данные с сайта"
		} else {
			var dataS SessionsData                                // структура SessionsData
			err := json.Unmarshal([]byte(responseString), &dataS) // декодируем JSON файл
			if err != nil {
				log.Println("[ERROR] SessionsData unmarshal error: ", err)
			}
			log.Printf("[INFO] Отключение %s\n", dataS.Sessions[0].CreatorIp)
			game, _ := readConfig(dataS.Sessions[0].ProductId, fileGames)
			billing := dataS.Sessions[0].BillingType
			if sessionDur != "off" {
				var billingTrial string
				if a.cfg.TrialON {
					if billing == "trial" {
						sumTrial = getValueByKey(dataS.Sessions[0].CreatorIp)
						if sumTrial < 20 || !a.cfg.TrialBlock {
							ipTrial := dataS.Sessions[0].CreatorIp
							handshake := dataS.Sessions[0].AbortComment
							if !strings.Contains(handshake, "handshake") { // если кнопка "Играть тут" активирована, добавляем время в файл
								createOrUpdateKeyValue(ipTrial, minute)
							}
							sumTrial = getValueByKey(dataS.Sessions[0].CreatorIp)
							billingTrial = fmt.Sprintf("\nTrial %dмин", sumTrial)
						} else if sumTrial > 20 && a.cfg.TrialBlock {
							billingTrial = fmt.Sprintf("\nKICK - Trial %dмин", sumTrial)
						}
					}
				}
				var comment string
				if dataS.Sessions[0].AbortComment != "" {
					comment = "\n" + dataS.Sessions[0].AbortComment
				}
				game = fmt.Sprintf("<b><i> %s </i></b>", game)
				if !a.cfg.StartMessageON {
					if a.cfg.OnlineIpInfo {
						ipInfo = onlineDBip(dataS.Sessions[0].CreatorIp)
					} else {
						ipInfo = offlineDBip(dataS.Sessions[0].CreatorIp)
					}
					infoString = "<b>🔴</b>" + hname + game + "\n" + sessionDur + "\n" + dataS.Sessions[0].CreatorIp + ipInfo + "\n" + comment + billingTrial + "\n" + serverIP
				} else {
					infoString = "<b>🔴</b>" + hname + game + "\n" + dataS.Sessions[0].CreatorIp + " - " + sessionDur + comment + billingTrial
				}

			} else {
				infoString = "off"
			}
		}
	} else if status == "Comment" { // проверяем написание коммента
		var sessionDur, commentC, game string
		var stopTime, startTime time.Time
		var dataC SessionsData
		for i := 0; i < 18; i++ {
			responseString, err := a.getFromURL(UrlSessions, "uuid", sessionId)
			if err != nil {
				infoString = hname + "невозможно получить данные с сайта"
				log.Println("[ERROR] Невозможно получить данные с сайта")
			} else {
				err := json.Unmarshal([]byte(responseString), &dataC) // декодируем JSON файл
				if err != nil {
					log.Println("[ERROR] SessionsData unmarshal error: ", err)
				}
				if dataC.Sessions[0].Comment == "" {
					time.Sleep(10 * time.Second)
				} else {
					log.Printf("[INFO] Отключение %s\n", dataC.Sessions[0].CreatorIp)
					game, _ = readConfig(dataC.Sessions[0].ProductId, fileGames)
					_, stopTime = dateTimeS(dataC.Sessions[0].FinishedOn)
					_, startTime = dateTimeS(dataC.Sessions[0].CreatedOn)
					sessionDur, _ = dur(stopTime, startTime)
					commentC = dataC.Sessions[0].Comment
					log.Printf("[INFO] Получение комментария %s\n, %s ", dataC.Sessions[0].CreatorIp, sessionId)
					infoString = "<b>🟡</b>" + hname + " - " + "<b><i>" + game + "</i></b>" + "\n" + dataC.Sessions[0].CreatorIp + " - " + sessionDur + "\n" + commentC
					i = 18
				}
			}
		}
	}
	return infoString
}

func runCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
