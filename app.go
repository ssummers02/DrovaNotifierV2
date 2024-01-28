package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/onrik/yaconf"
	log "github.com/sirupsen/logrus"
)

type App struct {
	cfg    Config
	tg     *TgClient
	appDir string
	dc     *DrovaClient
}

func NewApp() (*App, error) {
	stationName, err := os.Hostname()
	if err != nil {
		log.Println("[ERROR] Ошибка при получении имени компьютера: ", err)
		return nil, fmt.Errorf("[ERROR] Ошибка при получении имени компьютера: %v", err)
	}

	cfg := Config{}
	err = yaconf.Read("config.yml", &cfg)
	if err != nil {
		log.Println("[ERROR] Ошибка при чтении конфигурационного файла: ", err)
		return nil, fmt.Errorf("[ERROR] Ошибка при чтении конфигурационного файла: %v", err)
	}

	regFolder := `SOFTWARE\ITKey\Esme`
	serverID, err := regGet(regFolder, "last_server") // получаем ID сервера
	if err != nil {
		log.Println("[ERROR] Ошибка чтения ключа реестра: ", err)
		return nil, fmt.Errorf("[ERROR] Ошибка чтения ключа реестра: %v", err)
	}
	regFolder += `\servers\` + serverID
	authToken, err := regGet(regFolder, "auth_token") // получаем токен для авторизации
	if authToken == "" {
		log.Println("[ERROR] Ошибка чтения ключа реестра: ", err)
		return nil, fmt.Errorf("[ERROR] Ошибка чтения ключа реестра: %v", err)
	}

	go validToken(regFolder, authToken)

	cfg.hostName = stationName
	cfg.serverID = serverID
	cfg.authToken = authToken

	tg, err := NewTgClient(cfg.BotToken, cfg.ChatID, cfg.viewHostname, cfg.hostName, cfg.UserID, cfg.serverID)
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Ошибка подключения бота: %v", err)
	}

	// Получаем текущую директорию программы
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, fmt.Errorf("[ERROR] Ошибка получения текущей деректории: %v", err)
	}

	dc := NewDrovaClient()

	return &App{
		cfg:    cfg,
		tg:     tg,
		appDir: dir,
		dc:     dc,
	}, nil
}

func (a *App) Start() {
	go func() {
		err := a.dc.GetGame(a.appDir)
		if err != nil {
			log.Println("[ERROR] Ошибка при получении списка игр: ", err)
			return
		}
	}()
	go func() {
		err := a.antiCheat()
		if err != nil {
			log.Println("[ERROR] Ошибка при проверке файлов античитов: ", err)
			return
		}
	}() // проверка античитов
	go func() {
		err := a.diskSpace()
		if err != nil {
			log.Println("[ERROR] Ошибка при проверке свободного места на дисках: ", err)
			return
		}
	}() // проверка свободного места на дисках
	go func() {
		err := a.messageStartWin()
		if err != nil {
			log.Println("[ERROR] Ошибка при получении времени запуска windows: ", err)
			return
		}
	}() // проверка времени запуска станции

	go a.esmeCheck() // запуск мониторинга сервиса дров

	if a.cfg.CommandON {
		go a.commandBot()
	}

	if a.cfg.TrialON {
		log.Println("[INFO] Запись триала в ", filepath.Join(a.appDir, "trial.txt"))
	}

	if !a.cfg.OnlineIpInfo {
		if !a.cfg.AutoUpdateGeolite { // если не включен автоапдейт
			go restartGeoLite(filepath.Join(a.appDir, a.cfg.mmdbASN), filepath.Join(a.appDir, a.cfg.mmdbCity)) // запускаем проверку изменений файлов GeoLite
		} else { // иначе
			updateGeoLite(filepath.Join(a.appDir, a.cfg.mmdbASN), filepath.Join(a.appDir, a.cfg.mmdbCity)) // проверяем есть ли обновление для GeoLite
		}
	}

	if a.cfg.CheckTempON {
		isRunLibreHardwareMonitor, err := checkIfProcessRunning("LibreHardwareMonitor.exe")
		if err != nil {
			log.Println("[ERROR] Ошибка при проверке запущенных процессов: ", err)
			return
		}
		if !isRunLibreHardwareMonitor {
			log.Println("[ERROR] LibreHardwareMonitor.exe не запущен")
			return
		}
		go a.CheckHWt() // мониторинг температур

	}

}
