package main

import (
	"log"
	"os"
	"strconv"
)

var ( // true - включение функции, false - выключение
	BotToken      string        // токен бота
	Chat_IDint    int64         // определяем ID чата получателя
	UserID        int64         // ID пользователя, от которого принимаются команды
	ServiceChatID int64         // чат для сервисных сообщений
	CommandON     bool   = true // включить команды управления ботом

	OnlineIpInfo      bool = true // инфо по IP online
	AutoUpdateGeolite bool = true // автообновление файлов GeoLite с Github

	CheckAntiCheat bool    = true // проверка наличия файлов EasyAntiCheat.exe и EasyAntiCheat_EOS.exe
	CheckFreeSpace bool    = true // проверка свободного места на дисках
	CheckTempON    bool    = true // мониторинг температур
	FANt           float64 = 75   // порог проверки работы вентиляторов видеокарты
	FANrpm         float64 = 1000 // минимальные обороты при FANt
	CPUtmax        float64 = 85   // порог температуры процессора
	GPUtmax        float64 = 85   // порог температуры ядра видеокарты
	GPUhsTmax      float64 = 90   // порог температуры HotSpot видеокарты
	DeltaT         float64 = 5    // дельта среднего значения температур от от порога предупреждения. Для сообщения о нормализации температур

	TrialON      bool   = false // сбор статистики по триальщикам в trial.txt. false - не собирается статистика в trial.txt
	TrialBlock   bool   = false // Блокировка "хитрых" триальщиков. false - нет блокировки
	TrialfileLAN string = ``    // файл в сети пример `S:\trial.txt`

	StartMessageON   bool = true // включение сообщений при начале сессии. false - сообщение не будет приходить
	StopMessageON    bool = true // включение о сообщении об окончании сессии. false - сообщение не будет приходить
	ShortSessionON   bool = true // оповещать о сессиях менее Х минут, выставлять ниже. false - сообщение не будет приходить
	minMinute        int  = 10   // выставляем порог отправки сообщений о сессии. значения от 0 до 59
	CommentMessageON bool = true // сообщение с комментарием клиента. false - сообщение не будет приходить
)

func getConfigBot() (BotToken string, Chat_IDint, UserID, serviceChatID int64) {
	BotToken = "34355345:sdfasdasd" // "enter_your_bot_toket"
	Chat_IDint = 34355345           // чат, куда будут приходить информация
	UserID = 34355345               // пользователь, от которого будут приниматься команды
	ServiceChatID = 0               // чат для сервисных сообщений, 0 - отправка в Chat_IDint
	return BotToken, Chat_IDint, UserID, serviceChatID
}

func getConfigFile(fileConfig string) {
	//блок для получения данных из конфига
	_, err := os.Stat(fileConfig)
	if os.IsNotExist(err) {
		// Файл не существует
		log.Printf("[INFO] Файл %s отсутствует\n", fileConfig)
	} else {

		bToken, err := readConfig("tokenbot", fileConfig) // определяем токен бота
		if err != nil {
			log.Printf("[ERROR] Ошибка - %s. %s\n", err, getLine())
		}
		if bToken != "" {
			BotToken = bToken // получаем токен этого бота
		}

		ChatIDint := takeConfInt("chatID", fileConfig)
		if ChatIDint != 0 {
			Chat_IDint = ChatIDint
		}

		SChatID := takeConfInt("ServiceChatID", fileConfig)
		if SChatID != 0 {
			ServiceChatID = SChatID
		}

		UID := takeConfInt("UserID", fileConfig)
		if UID != 0 {
			UserID = UID
		}

		OnlineIpInfo = takeConfBool("onlineIpInfo") // настройки получения инфо по IP
		log.Println("OnlineIpInfo - ", OnlineIpInfo)
		CheckFreeSpace = takeConfBool("checkFreeSpace") // проверка свободного места на дисках
		log.Println("CheckFreeSpace - ", CheckFreeSpace)
		CheckAntiCheat = takeConfBool("checkAntiCheat") // проверка папок античитов
		log.Println("CheckAntiCheat - ", CheckAntiCheat)
		CommandON = takeConfBool("CommandON") // управление ботом через чат ТГ
		log.Println("CommandON - ", CommandON)
		StartMessageON = takeConfBool("StartMessageON") // включить сообщения о начале сессии
		log.Println("StartMessageON - ", StartMessageON)
		StopMessageON = takeConfBool("StopMessageON") // включить сообщения об окончании сессии
		log.Println("StopMessageON - ", StopMessageON)
		ShortSessionON = takeConfBool("ShortSessionON") // включить короткие сообщения
		log.Println("ShortSessionON - ", ShortSessionON)
		CommentMessageON = takeConfBool("CommentMessageON") // включить комментарии
		log.Println("CommentMessageON - ", CommentMessageON)

		CheckTempON = takeConfBool("CheckTempON") // мониторинг температур
		log.Println("CheckTempON - ", CheckTempON)
		if CheckTempON {
			FANt = takeConfFloat("FANt", fileConfig) // порог проверки работы вентиляторов видеокарты
			log.Println("FANt - ", FANt)
			FANrpm = takeConfFloat("FANrpm", fileConfig) // минимальные обороты при FANt
			log.Println("FANrpm - ", FANrpm)
			CPUtmax = takeConfFloat("CPUtmax", fileConfig) // порог температуры процессора
			log.Println("CPUtmax - ", CPUtmax)
			GPUtmax = takeConfFloat("GPUtmax", fileConfig) // порог температуры ядра видеокарты
			log.Println("GPUtmax - ", GPUtmax)
			GPUhsTmax = takeConfFloat("GPUhsTmax", fileConfig) // порог температуры HotSpot видеокарты
			log.Println("GPUhsTmax - ", GPUhsTmax)
		}

		TrialON = takeConfBool("TrialON") // вести статистику триала

		if TrialON {
			TrialBlock = takeConfBool("trialBlock")                    // блокировка триальщиков
			TrialfileLAN, err = readConfig("TrialfileLAN", fileConfig) // определяем токен бота
			if err != nil {
				log.Printf("[ERROR] Ошибка - %s. %s\n", err, getLine())
			}
		}
	}
}

func takeConfInt(param, file string) (paramInt int64) {
	paramValue, err := readConfig(param, file) // определяем ID чата
	if err != nil {
		log.Printf("[ERROR] ServiceChatID - %s. %s\n", err, getLine())
	}
	if paramValue != "" {
		paramInt, err = strconv.ParseInt(paramValue, 10, 64) // конвертируем ID чата в int64
		if err != nil {
			log.Printf("[ERROR] %s:  %s. %s\n", paramValue, err, getLine())
		}
	}
	return paramInt
}

// получение данны из файла конфига
func takeConfBool(key string) (value bool) {
	check, err := readConfig(key, fileConfig)
	if err != nil {
		log.Printf("[ERROR] Ошибка - %s. %s\n", err, getLine())
	}
	if check == "true" {
		value = true
	} else {
		value = false
	}
	return
}

func takeConfFloat(param, file string) (paramFloat float64) {
	paramValue, err := readConfig(param, file) // определяем ID чата
	if err != nil {
		log.Printf("[ERROR] ServiceChatID - %s. %s\n", err, getLine())
	}
	if paramValue != "" {
		paramFloat, err = strconv.ParseFloat(paramValue, 64) // конвертируем ID чата в int64
		if err != nil {
			log.Printf("[ERROR] %s:  %s. %s\n", paramValue, err, getLine())
		}
	}
	return paramFloat
}
