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
		Client_id     string `json:"client_id"`
		Product_id    string `json:"product_id"`
		Created_on    int64  `json:"created_on"`
		Finished_on   int64  `json:"finished_on"` //or null
		Status        string `json:"status"`
		Creator_ip    string `json:"creator_ip"`
		Abort_comment string `json:"abort_comment"` //or null
		Billing_type  string `json:"billing_type"`  // or null
	}
}

func sessionInfo(status string) (infoString string) {
	var sumTrial int

	responseString := getFromURL(UrlSessions, serverID)

	var data SessionsData                         // структура SessionsData
	json.Unmarshal([]byte(responseString), &data) // декодируем JSON файл

	if status == "Start" { // формируем текст для отправки
		var serverIP string

		game, _ := readConfig(data.Sessions[0].Product_id, fileGames)
		sessionOn, _ := dateTimeS(data.Sessions[0].Created_on)
		ipInfo = ""

		if onlineIpInfo {
			ipInfo = onlineDBip(data.Sessions[0].Creator_ip)
		} else {
			ipInfo = offlineDBip(data.Sessions[0].Creator_ip)
		}
		var billing string
		billing = data.Sessions[0].Billing_type
		if billing != "" && billing != "trial" {
			billing = " - " + data.Sessions[0].Billing_type
		}
		if billing == "trial" {
			sumTrial = getValueByKey(data.Sessions[0].Creator_ip)
			if sumTrial == -1 { // нет записей по этому IP
				createOrUpdateKeyValue(data.Sessions[0].Creator_ip, 0)
				billing = " - " + data.Sessions[0].Billing_type
			} else if sumTrial >= 0 && sumTrial < 19 { // уже подключался, но не играл в общей сложности 19 минуту
				billing = fmt.Sprintf(" - TRIAL %dмин", sumTrial)
			} else if sumTrial > 18 { // начал злоупотреблять
				billing = fmt.Sprintf(" - TRIAL %dмин\nЗлоупотребление Триалом!", sumTrial)

				if trialBlock {
					text := "Злоупотребление Триалом! Кикаем!"
					message := fmt.Sprintf("Внимание! Станция %s.\n%s", hostname, text)
					err := SendMessage(BotToken, Chat_IDint, message)
					if err != nil {
						log.Println("Ошибка отправки сообщения: ", err, getLine())
					}
					log.Printf("Заблокировано соединение: %s. Trial %d", data.Sessions[0].Creator_ip, sumTrial)
					time.Sleep(10 * time.Second)
					err = runCommand("taskkill", "/IM", "ese.exe", "/F")
					if err != nil {
						fmt.Println("Ошибка выполнения команды:", err)
						return
					}
				}
			}
		}
		localAddr, nameInterface := getInterface()
		serverIP = "\n" + nameInterface + " - " + localAddr
		infoString = "[+]" + hostname + " - " + game + "\n" + data.Sessions[0].Creator_ip + ipInfo + "\n" + sessionOn + billing + serverIP

	} else if status == "Stop" { // высчитываем продолжительность сессии и формируем текст для отправки
		var minute int
		var duration, sessionDur string
		game, _ := readConfig(data.Sessions[0].Product_id, fileGames)

		_, stopTime := dateTimeS(data.Sessions[0].Finished_on)
		_, startTime := dateTimeS(data.Sessions[0].Created_on)
		if data.Sessions[0].Created_on < data.Sessions[0].Finished_on {
			duration, minute = dur(stopTime, startTime)
			sessionDur = " - " + duration
		}
		billing := data.Sessions[0].Billing_type

		var billingTrial string
		billingTrial = ""
		if billing == "trial" {
			sumTrial = getValueByKey(data.Sessions[0].Creator_ip)
			if sumTrial < 20 || !trialBlock {
				ipTrial := data.Sessions[0].Creator_ip
				handshake := data.Sessions[0].Abort_comment
				if !strings.Contains(handshake, "handshake") { // если кнопка "Играть тут" активированна, добавляем время в файл
					createOrUpdateKeyValue(ipTrial, minute)
				}
				sumTrial = getValueByKey(data.Sessions[0].Creator_ip)
				billingTrial = fmt.Sprintf("\nTrial %dмин", sumTrial)
			} else if sumTrial > 20 && trialBlock {
				billingTrial = fmt.Sprintf("\nKICK - Trial %dмин", sumTrial)
				sessionDur = ""
			}
		}
		var comment string
		if data.Sessions[0].Abort_comment != "" {
			comment = "\n" + data.Sessions[0].Abort_comment
		}
		infoString = "[-]" + hostname + " - " + game + "\n" + data.Sessions[0].Creator_ip + sessionDur + comment + billingTrial
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
