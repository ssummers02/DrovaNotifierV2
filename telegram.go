package main

import (
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type TgClient struct {
	*tgbotapi.BotAPI
	chatID       int64
	viewHostname bool
	hostName     string
	userID       int64
	serverID     string
}

func NewTgClient(botToken string, chatID int64, viewHostname bool, hostName string, userID int64, serverID string) (*TgClient, error) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Ошибка подключения бота: %v", err)
	}

	return &TgClient{
		BotAPI:       bot,
		chatID:       chatID,
		viewHostname: viewHostname,
		hostName:     hostName,
		userID:       userID,
		serverID:     serverID,
	}, nil
}

// отправка сообщения ботом
func (t *TgClient) SendMessage(text string) error {
	var hname = ""
	if t.viewHostname {
		hname = fmt.Sprintf(" Станция %s\n", t.hostName)
	}

	message := tgbotapi.NewMessage(t.chatID, hname+text)
	message.ParseMode = "HTML"

	for i := 0; i < 3; i++ {
		_, err := t.Send(message)
		if err != nil {
			log.Println("[ERROR] Ошибка отправки сообщения: ", err)
			time.Sleep(3 * time.Second)
			return err
		}
	}

	return nil
}

func (t *TgClient) commandBot() {
	var messageT, honame, hname string

	// таймаут обновления бота
	upd := tgbotapi.NewUpdate(0)
	upd.Timeout = 60

	// получаем обновления от API
	updates := t.GetUpdatesChan(upd)

	for update := range updates {
		//проверяем тип обновления - только новые входящие сообщения
		if update.Message != nil {

			if update.Message.From.ID == t.userID {
				messageT = strings.ToLower(update.Message.Text)

				if strings.Contains(messageT, "/reboot") {
					if strings.Contains(messageT, honame) { // Проверяем, что в тексте упоминается имя ПК
						log.Println("Перезагрузка ПК по команде из телеграмма")
						message := fmt.Sprintf("Станция %s будет перезагружена по команде из телеграмма", t.hostName)
						err := t.SendMessage(message)
						if err != nil {
							log.Println("[ERROR] Ошибка отправки сообщения: ", err)
							return
						}
						rebootPC()
					} else {
						anotherPC(t.hostName)
					}
				} else if strings.Contains(messageT, "/status") {
					var serv serverManager // структура serverManager
					responseData, err := getFromURL(UrlServers, "server_id", t.serverID)
					log.Println("получили команду /статус")
					if err != nil {
						chatMessage := t.hostName + " Невозможно получить данные с сайта"
						log.Println("[ERROR] Невозможно получить данные с сайта")
						err := t.SendMessage(chatMessage) // отправка сообщения
						if err != nil {
							log.Println("[ERROR] Ошибка отправки сообщения: ", err)
						}
					} else {
						json.Unmarshal([]byte(responseData), &serv) // декодируем JSON файл

						var serverName, status, messageText string
						messageText = ""
						i := 0
						// messageText = fmt.Sprint(hname)
						// log.Println("/статус - ошибок нет, собираем данные")
						for range serv {
							// log.Println("/статус - зашли в рэндж серверов")
							var sessionStart, server_ID string
							serverName = serv[i].Name
							status = serv[i].Status // Получаем статус сервера
							server_ID = serv[i].Server_id
							// log.Println(serverName, "-", status, "-", server_ID)

							if status == "BUSY" || status == "HANDSHAKE" { // Получаем время начала, если станция занят
								var data SessionsData // структура SessionsData
								responseData, err := getFromURL(UrlSessions, "server_id", server_ID)
								if err != nil {
									chatMessage := t.hostName + " Невозможно получить данные с сайта"
									log.Println("[ERROR] Невозможно получить данные с сайта")
									err := t.SendMessage(chatMessage) // отправка сообщения
									if err != nil {
										log.Println("[ERROR] Ошибка отправки сообщения: ", err)
									}
									sessionStart = ""
								} else {
									json.Unmarshal([]byte(responseData), &data) // декодируем JSON файл
									startTime, _ := dateTimeS(data.Sessions[0].Created_on)
									sessionStart = fmt.Sprintf("\n%s", startTime)
								}
							} else {
								sessionStart = ""
							}
							messageText += fmt.Sprintf("%s - %s%s\n\n", serverName, status, sessionStart)
							i++
						}

						err := t.SendMessage(messageText)
						if err != nil {
							log.Println("[ERROR] Ошибка отправки сообщения: ", err)
							return
						}
					}
				} else if strings.Contains(messageT, "/visible") {
					if strings.Contains(messageT, honame) { // Проверяем, что в тексте упоминается имя ПК
						err := viewStation("true", t.serverID)
						if err != nil {
							log.Println("[ERROR] Ошибка смены статуса: ", err)
							message := fmt.Sprintf("Ошибка. Станция %s не видна клиентам. Повторите попытку позже", t.hostName)
							err = t.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						} else {
							log.Printf("Станция %s в сети\n", t.hostName)
							message := fmt.Sprintf("Станция %s видна клиентам", t.hostName)
							err = t.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						}
					} else {
						anotherPC(t.hostName)
					}
				} else if strings.Contains(messageT, "/invisible") {
					if strings.Contains(messageT, honame) { // Проверяем, что в тексте упоминается имя ПК
						err := viewStation("false", t.serverID)
						if err != nil {
							log.Println("[ERROR] Ошибка смены статуса: ", err)
							message := fmt.Sprintf("Ошибка. Станция %s не спрятана от клиентов. Повторите попытку позже", t.hostName)
							err = t.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						} else {
							log.Printf("Станция %s спрятана\n", t.hostName)
							message := fmt.Sprintf("Станция %s спрятана от клиентов", t.hostName)
							err = t.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						}
					} else {
						anotherPC(t.hostName)
					}
				} else if strings.Contains(messageT, "/temp") {
					log.Println("Получение температур и оборотов вентиляторов")
					var message string
					_, _, _, _, _, _, _, message = GetTemperature()

					message = hname + message
					err := t.SendMessage(message)
					if err != nil {
						log.Println("[ERROR] Ошибка отправки сообщения: ", err)
						return
					}
				} else if strings.Contains(messageT, "/delayreboot") {
					if strings.Contains(messageT, honame) { // Проверяем, что в тексте упоминается имя ПК
						go delayReboot(0)
						message := fmt.Sprintf("Будет выполнена перезагрузка %sпо окончании сессии", hname)
						err := t.SendMessage(message)
						if err != nil {
							log.Println("[ERROR] Ошибка отправки сообщения: ", err)
							return
						}
					} else {
						anotherPC(t.hostName)
					}
				} else if strings.Contains(messageT, "/drovastop") {
					if strings.Contains(messageT, honame) { // Проверяем, что в тексте упоминается имя ПК
						err := drovaService("stop")
						if err != nil {
							message := fmt.Sprintf("%sОшибка завершения задачи Streaming Service", hname)
							err := t.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						} else {
							message := fmt.Sprintf("%sЗадача Streaming Service остановлена", hname)
							err := t.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						}
					} else {
						anotherPC(t.hostName)
					}
				} else if strings.Contains(messageT, "/drovastart") {
					if strings.Contains(messageT, honame) { // Проверяем, что в тексте упоминается имя ПК
						err := drovaService("start")
						if err != nil {
							message := fmt.Sprintf("%sОшибка запуска задачи Streaming Service", hname)
							err := t.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						} else {
							message := fmt.Sprintf("%sЗадача Streaming Service запущена", hname)
							err := t.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						}
					} else {
						anotherPC(t.hostName)
					}
				} else if strings.Contains(messageT, "/start") {
					message := fmt.Sprintln("Доступные комманды. ST1 имя вашего ПК")
					message += fmt.Sprintln("/rebootST1 - перезагрузить ST1")
					message += fmt.Sprintln("/delayrebootST1 - перезагрузка ST1 когда закончится сессия")
					message += fmt.Sprintln("/visibleST1 - скрыть ST1")
					message += fmt.Sprintln("/invisibleST1 - скрыть ST1")
					message += fmt.Sprintln("/status - статус серверов")
					message += fmt.Sprintln("/temp - температуры")
					// message += fmt.Sprintln("/drovastartST1 - старт Streaming Service ST1")
					// message += fmt.Sprintln("/drovastopST1 - стоп Streaming Service ST1")
					// message += fmt.Sprintln("")
					// message += fmt.Sprintln("")

					err := t.SendMessage(message)
					if err != nil {
						log.Println("[ERROR] Ошибка отправки сообщения: ", err)
						return
					}
				} else {
					messageText := "Неизвестная команда"
					err := t.SendMessage(messageText)
					if err != nil {
						log.Println("[ERROR] Ошибка отправки сообщения: ", err)
						return
					}
				}
			}
			log.Printf("Сообщение от %d: %s", update.Message.From.ID, update.Message.Text)
		}
	}
}
