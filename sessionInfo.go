package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// структура для выгрузки информации по сессиям
type SessionsData struct {
	Sessions []struct {
		Session_uuid  string `json:"uuid"`
		Client_id     string `json:"client_id"`
		Product_id    string `json:"product_id"`
		Created_on    int64  `json:"created_on"`
		Finished_on   int64  `json:"finished_on"` //or null
		Status        string `json:"status"`
		Creator_ip    string `json:"creator_ip"`
		Abort_comment string `json:"abort_comment"` //or null
		Score         string `json:"score"`         //or null
		ScoreReason   string `json:"score_reason"`  //or null
		Comment       string `json:"score_text"`    //or null
		Billing_type  string `json:"billing_type"`  // or null
	}
}

func sessionInfo(status string) (infoString string) {
	var sumTrial int
	var serverIP string
	if status == "Start" { // формируем текст для отправки
		responseString := getFromURL(UrlSessions, "server_id", serverID)

		var data SessionsData                         // структура SessionsData
		json.Unmarshal([]byte(responseString), &data) // декодируем JSON файл
		Session_ID = data.Sessions[0].Session_uuid

		log.Printf("[INFO] Подключение %s, billing: %s\n", data.Sessions[0].Creator_ip, data.Sessions[0].Billing_type)
		game, _ := readConfig(data.Sessions[0].Product_id, fileGames)
		sessionOn, _ := dateTimeS(data.Sessions[0].Created_on)
		ipInfo = ""

		if OnlineIpInfo {
			ipInfo = onlineDBip(data.Sessions[0].Creator_ip)
		} else {
			ipInfo = offlineDBip(data.Sessions[0].Creator_ip)
		}
		var billing string
		billing = data.Sessions[0].Billing_type
		if billing != "" && billing != "trial" {
			billing = " - " + data.Sessions[0].Billing_type
		}
		if TrialON {
			if billing == "trial" {
				sumTrial = getValueByKey(data.Sessions[0].Creator_ip)
				if sumTrial == -1 { // нет записей по этому IP
					createOrUpdateKeyValue(data.Sessions[0].Creator_ip, 0)
					billing = " - " + data.Sessions[0].Billing_type
				} else if sumTrial >= 0 && sumTrial < 19 { // уже подключался, но не играл в общей сложности 19 минуту
					billing = fmt.Sprintf(" - TRIAL %dмин", sumTrial)
				} else if sumTrial > 18 { // начал злоупотреблять
					billing = fmt.Sprintf(" - TRIAL %dмин\nЗлоупотребление Триалом!", sumTrial)

					if TrialBlock {
						text := "Злоупотребление Триалом! Кикаем!"
						message := fmt.Sprintf("Внимание! Станция %s.\n%s", hostname, text)
						err := SendMessage(BotToken, Chat_IDint, message)
						if err != nil {
							log.Println("[ERROR] Ошибка отправки сообщения: ", err, getLine())
						}
						log.Printf("[INFO] Заблокировано соединение: %s. Trial %d", data.Sessions[0].Creator_ip, sumTrial)
						time.Sleep(10 * time.Second)
						err = runCommand("taskkill", "/IM", "ese.exe", "/F")
						if err != nil {
							log.Println("[ERORR] Ошибка выполнения команды:", err)
							return
						}
					}
				}
			}
		}
		localAddr, nameInterface := getInterface()
		serverIP = "\n" + nameInterface + " - " + localAddr
		infoString = "[+]" + hostname + " - " + game + "\n" + data.Sessions[0].Creator_ip + ipInfo + "\n" + sessionOn + billing + serverIP

	} else if status == "Stop" { // высчитываем продолжительность сессии и формируем текст для отправки
		responseString := getFromURL(UrlSessions, "uuid", Session_ID)

		var data SessionsData                         // структура SessionsData
		json.Unmarshal([]byte(responseString), &data) // декодируем JSON файл

		var minute int
		var sessionDur string
		log.Printf("[INFO] Отключение %s\n", data.Sessions[0].Creator_ip)
		time.Sleep(10 * time.Second)
		game, _ := readConfig(data.Sessions[0].Product_id, fileGames)

		_, stopTime := dateTimeS(data.Sessions[0].Finished_on)
		// log.Println("[DIAG]data.Sessions[0].Finished_on - ", data.Sessions[0].Finished_on)
		// log.Println("[DIAG]stopTime - ", stopTime)
		_, startTime := dateTimeS(data.Sessions[0].Created_on)
		// log.Println("[DIAG]data.Sessions[0].Created_on - ", data.Sessions[0].Created_on)
		// log.Println("[DIAG]startTime - ", startTime)
		if data.Sessions[0].Created_on < data.Sessions[0].Finished_on {
			sessionDur, minute = dur(stopTime, startTime)
		}
		billing := data.Sessions[0].Billing_type
		if sessionDur != "off" {
			var billingTrial string = ""
			if TrialON {
				if billing == "trial" {
					sumTrial = getValueByKey(data.Sessions[0].Creator_ip)
					if sumTrial < 20 || !TrialBlock {
						ipTrial := data.Sessions[0].Creator_ip
						handshake := data.Sessions[0].Abort_comment
						if !strings.Contains(handshake, "handshake") { // если кнопка "Играть тут" активированна, добавляем время в файл
							createOrUpdateKeyValue(ipTrial, minute)
						}
						sumTrial = getValueByKey(data.Sessions[0].Creator_ip)
						billingTrial = fmt.Sprintf("\nTrial %dмин", sumTrial)
					} else if sumTrial > 20 && TrialBlock {
						billingTrial = fmt.Sprintf("\nKICK - Trial %dмин", sumTrial)
						// sessionDur = ""
					}
				}
			}
			var comment string
			if data.Sessions[0].Abort_comment != "" {
				comment = "\n" + data.Sessions[0].Abort_comment
			}
			if !StartMessageON {
				if OnlineIpInfo {
					ipInfo = onlineDBip(data.Sessions[0].Creator_ip)
				} else {
					ipInfo = offlineDBip(data.Sessions[0].Creator_ip)
				}
				infoString = "[-]" + hostname + " - " + game + "\n" + sessionDur + "\n" + data.Sessions[0].Creator_ip + ipInfo + "\n" + comment + billingTrial + "\n" + serverIP
			} else {
				infoString = "[-]" + hostname + " - " + game + "\n" + data.Sessions[0].Creator_ip + " - " + sessionDur + comment + billingTrial
			}

		} else {
			infoString = "off"
		}
	} else if status == "Comment" { // проверяем написание коммента
		session_ID := Session_ID
		time.Sleep(2 * time.Minute)
		responseString := getFromURL(UrlSessions, "uuid", session_ID)

		var data SessionsData                         // структура SessionsData
		json.Unmarshal([]byte(responseString), &data) // декодируем JSON файл

		if data.Sessions[0].Comment != "" {

			// var minute int
			var sessionDur, commentC string
			log.Printf("[INFO] Получение комментария %s\n, %s ", data.Sessions[0].Creator_ip, session_ID)
			game, _ := readConfig(data.Sessions[0].Product_id, fileGames)
			_, stopTime := dateTimeS(data.Sessions[0].Finished_on)
			_, startTime := dateTimeS(data.Sessions[0].Created_on)
			sessionDur, _ = dur(stopTime, startTime)

			// billing := data.Sessions[0].Billing_type
			// var score, scoreReason, commentC string
			// score = data.Sessions[0].Score
			// scoreReason = data.Sessions[0].ScoreReason
			commentC = data.Sessions[0].Comment

			// if OnlineIpInfo {
			// 	ipInfo = onlineDBip(data.Sessions[0].Creator_ip)
			// } else {
			// 	ipInfo = offlineDBip(data.Sessions[0].Creator_ip)
			// }
			infoString = hostname + " - " + game + "\n" + data.Sessions[0].Creator_ip + " - " + sessionDur + "\n" + commentC

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
