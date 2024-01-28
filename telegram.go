package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	log "github.com/sirupsen/logrus"
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

// SendMessage отправка сообщения ботом
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

func (a *App) commandBot() {
	var messageT, honame, hname string

	// таймаут обновления бота
	upd := tgbotapi.NewUpdate(0)
	upd.Timeout = 60

	// получаем обновления от API
	updates := a.tg.GetUpdatesChan(upd)

	for update := range updates {
		//проверяем тип обновления - только новые входящие сообщения
		if update.Message != nil {

			if update.Message.From.ID == a.tg.userID {
				messageT = strings.ToLower(update.Message.Text)

				if strings.Contains(messageT, "/reboot") {
					if strings.Contains(messageT, honame) { // Проверяем, что в тексте упоминается имя ПК
						log.Println("Перезагрузка ПК по команде из телеграмма")
						message := fmt.Sprintf("Станция %s будет перезагружена по команде из телеграмма", a.tg.hostName)
						err := a.tg.SendMessage(message)
						if err != nil {
							log.Println("[ERROR] Ошибка отправки сообщения: ", err)
							return
						}
						rebootPC()
					} else {
						a.anotherPC(a.tg.hostName)
					}
				} else if strings.Contains(messageT, "/status") {
					var serv serverManager // структура serverManager
					responseData, err := a.getFromURL(UrlServers, "server_id", a.tg.serverID)
					log.Println("получили команду /статус")
					if err != nil {
						chatMessage := a.tg.hostName + " Невозможно получить данные с сайта"
						log.Println("[ERROR] Невозможно получить данные с сайта")
						err := a.tg.SendMessage(chatMessage) // отправка сообщения
						if err != nil {
							log.Println("[ERROR] Ошибка отправки сообщения: ", err)
						}
					} else {
						err := json.Unmarshal([]byte(responseData), &serv) // декодируем JSON файл
						if err != nil {
							log.Println("[ERROR] serverManager unmarshal error: ", err)
						}
						var serverName, status, messageText string
						messageText = ""
						i := 0
						for range serv {
							var sessionStart, serverId string
							serverName = serv[i].Name
							status = serv[i].Status // Получаем статус сервера
							serverId = serv[i].ServerId

							if status == "BUSY" || status == "HANDSHAKE" { // Получаем время начала, если станция занят
								var data SessionsData // структура SessionsData
								responseData, err := a.getFromURL(UrlSessions, "server_id", serverId)
								if err != nil {
									chatMessage := a.tg.hostName + " Невозможно получить данные с сайта"
									log.Println("[ERROR] Невозможно получить данные с сайта")
									err := a.tg.SendMessage(chatMessage) // отправка сообщения
									if err != nil {
										log.Println("[ERROR] Ошибка отправки сообщения: ", err)
									}
									sessionStart = ""
								} else {
									err := json.Unmarshal([]byte(responseData), &data) // декодируем JSON файл
									if err != nil {
										log.Println("[ERROR] SessionsData unmarshal error: ", err)
									}
									startTime, _ := dateTimeS(data.Sessions[0].CreatedOn)
									sessionStart = fmt.Sprintf("\n%s", startTime)
								}
							} else {
								sessionStart = ""
							}
							messageText += fmt.Sprintf("%s - %s%s\n\n", serverName, status, sessionStart)
							i++
						}

						err = a.tg.SendMessage(messageText)
						if err != nil {
							log.Println("[ERROR] Ошибка отправки сообщения: ", err)
							return
						}
					}
				} else if strings.Contains(messageT, "/visible") {
					if strings.Contains(messageT, honame) { // Проверяем, что в тексте упоминается имя ПК
						err := a.viewStation("true", a.tg.serverID)
						if err != nil {
							log.Println("[ERROR] Ошибка смены статуса: ", err)
							message := fmt.Sprintf("Ошибка. Станция %s не видна клиентам. Повторите попытку позже", a.tg.hostName)
							err = a.tg.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						} else {
							log.Printf("Станция %s в сети\n", a.tg.hostName)
							message := fmt.Sprintf("Станция %s видна клиентам", a.tg.hostName)
							err = a.tg.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						}
					} else {
						a.anotherPC(a.tg.hostName)
					}
				} else if strings.Contains(messageT, "/invisible") {
					if strings.Contains(messageT, honame) { // Проверяем, что в тексте упоминается имя ПК
						err := a.viewStation("false", a.tg.serverID)
						if err != nil {
							log.Println("[ERROR] Ошибка смены статуса: ", err)
							message := fmt.Sprintf("Ошибка. Станция %s не спрятана от клиентов. Повторите попытку позже", a.tg.hostName)
							err = a.tg.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						} else {
							log.Printf("Станция %s спрятана\n", a.tg.hostName)
							message := fmt.Sprintf("Станция %s спрятана от клиентов", a.tg.hostName)
							err = a.tg.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						}
					} else {
						a.anotherPC(a.tg.hostName)
					}
				} else if strings.Contains(messageT, "/temp") {
					log.Println("Получение температур и оборотов вентиляторов")
					var message string
					_, _, _, _, _, _, _, message = GetTemperature()

					message = hname + message
					err := a.tg.SendMessage(message)
					if err != nil {
						log.Println("[ERROR] Ошибка отправки сообщения: ", err)
						return
					}
				} else if strings.Contains(messageT, "/delayreboot") {
					if strings.Contains(messageT, honame) { // Проверяем, что в тексте упоминается имя ПК
						go a.delayReboot(0)
						message := fmt.Sprintf("Будет выполнена перезагрузка %sпо окончании сессии", hname)
						err := a.tg.SendMessage(message)
						if err != nil {
							log.Println("[ERROR] Ошибка отправки сообщения: ", err)
							return
						}
					} else {
						a.anotherPC(a.tg.hostName)
					}
				} else if strings.Contains(messageT, "/drovastop") {
					if strings.Contains(messageT, honame) { // Проверяем, что в тексте упоминается имя ПК
						err := drovaService("stop")
						if err != nil {
							message := fmt.Sprintf("%sОшибка завершения задачи Streaming Service", hname)
							err := a.tg.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						} else {
							message := fmt.Sprintf("%sЗадача Streaming Service остановлена", hname)
							err := a.tg.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						}
					} else {
						a.anotherPC(a.tg.hostName)
					}
				} else if strings.Contains(messageT, "/drovastart") {
					if strings.Contains(messageT, honame) { // Проверяем, что в тексте упоминается имя ПК
						err := drovaService("start")
						if err != nil {
							message := fmt.Sprintf("%sОшибка запуска задачи Streaming Service", hname)
							err := a.tg.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						} else {
							message := fmt.Sprintf("%sЗадача Streaming Service запущена", hname)
							err := a.tg.SendMessage(message)
							if err != nil {
								log.Println("[ERROR] Ошибка отправки сообщения: ", err)
								return
							}
						}
					} else {
						a.anotherPC(a.tg.hostName)
					}
				} else if strings.Contains(messageT, "/start") {
					message := fmt.Sprintln("Доступные команды. ST1 имя вашего ПК")
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

					err := a.tg.SendMessage(message)
					if err != nil {
						log.Println("[ERROR] Ошибка отправки сообщения: ", err)
						return
					}
				} else {
					messageText := "Неизвестная команда"
					err := a.tg.SendMessage(messageText)
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
