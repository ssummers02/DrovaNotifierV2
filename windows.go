package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/StackExchange/wmi"
	"github.com/shirou/gopsutil/disk"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/windows/registry"
)

// проверка файлов античитов
func (a *App) antiCheat() error {
	if !a.cfg.CheckAntiCheat {
		return nil
	}

	antiCheatMap := map[string]string{
		"EasyAntiCheat_EOS": "C:\\Program Files (x86)\\EasyAntiCheat_EOS\\EasyAntiCheat_EOS.exe",
		"EasyAntiCheat":     "C:\\Program Files (x86)\\EasyAntiCheat\\EasyAntiCheat.exe",
	}

	for key, filePath := range antiCheatMap {
		if _, err := os.Stat(filePath); err != nil {
			message := fmt.Sprintf("[INFO] Внимание! Отсутствует файл %s", key)
			errMessage := a.tg.SendMessage(message)
			if err != nil {
				return fmt.Errorf("[ERROR] Ошибка отправки сообщения: %s", errMessage)
			}
			return fmt.Errorf("[INFO] Внимание!Отсутствует файл %s", key)
		}
	}

	log.Println("[INFO] Проверка файлов античитов: OK")
	return nil
}

// Проверяем свободное место на дисках
func (a *App) diskSpace() error {
	if !a.cfg.CheckFreeSpace {
		return nil
	}
	text := strings.Builder{}
	partitions, err := disk.Partitions(false)
	if err != nil {
		return fmt.Errorf("[ERROR] Ошибка получения данных о дисках: %v", err)
	}

	for _, partition := range partitions {
		usageStat, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			return fmt.Errorf("[ERROR] Ошибка получения данных для диска %s: %v", partition.Mountpoint, err)
		}

		freeSpace := float32(usageStat.Free) / (1024 * 1024 * 1024)
		if usageStat.UsedPercent > 90 {
			text.WriteString(fmt.Sprintf("На диске %s свободно менее 10%%, %.2f Гб\n", partition.Mountpoint, freeSpace))
		}
	}

	// Если text не пустой, значит есть диск со свободным местом менее 10%, отправляем сообщение
	if text.Len() > 0 {
		message := fmt.Sprintf("Внимание!\n%s", text.String())
		err := a.tg.SendMessage(message)
		if err != nil {
			return fmt.Errorf("[ERROR] Ошибка отправки сообщения: %s", err)
		}
	}

	return nil
}

// Оповещение о включении станции
func (a *App) messageStartWin() error {
	var osInfo []Win32Operatingsystem
	err := wmi.Query("SELECT LastBootUpTime FROM Win32_OperatingSystem", &osInfo)
	if err != nil {
		return fmt.Errorf("[ERROR] Ошибка получения данных о времени запуска Windows: %v", err)
	}

	lastBootUpTime := osInfo[0].LastBootUpTime
	formattedTime := lastBootUpTime.Format("02-01-2006 15:04:05")
	log.Println("[INFO] Windows запущен - ", formattedTime)
	// Получаем текущее время
	currentTime := time.Now()

	// Вычисляем разницу во времени
	duration := currentTime.Sub(lastBootUpTime)

	// Если прошло менее 5 минут с момента запуска Windows
	if duration.Minutes() < 5 {

		message := fmt.Sprintf("Внимание! Станция запущена менее 5 минут назад!\nВремя запуска - %s", formattedTime)
		err := a.tg.SendMessage(message)
		if err != nil {
			return fmt.Errorf("[ERROR] Ошибка отправки сообщения: %s", err)
		}
	}

	return nil
}

// Проверка на валидность токена
func validToken(regFolder, authToken string) {
	for {
		authTokenV, err := regGet(regFolder, "auth_token") // получаем токен для авторизации
		if err != nil {
			log.Println("[ERROR] Ошибка чтения ключа реестра: ", err)
			restart()
		}
		if authToken != authTokenV {
			log.Println("[INFO] Токен не совпадает, перезапуск приложения")
			restart()
		}
		time.Sleep(5 * time.Minute)
	}
}

// Получаем данные из реестра
func regGet(regFolder, keys string) (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, regFolder, registry.QUERY_VALUE)
	if err != nil {
		return "", fmt.Errorf("ошибка открытия ключа реестра: %v", err)
	}
	defer func(key registry.Key) {
		err := key.Close()
		if err != nil {
			log.Println("key.Close(): ", err)
		}
	}(key)

	value, _, err := key.GetStringValue(keys)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения ключа реестра: %v", err)
	}

	return value, nil
}

// Проверяем запущен ли Drova service
func (a *App) esmeCheck() {
	var i, y uint8 = 0, 0
	for {
		// Если процесс не запущен, с каждой следующей проверкой увеличиваем задержку отправки сообщения
		// используя переменную i. 2-е оповещение через 20минут после первого, 3-е через 30минут после второго
		// после отправки 3‑х сообщений, отправляем оповещение\напоминание с интервалом в 2часа
		if i < 3 {
			for y = 0; y <= i; y++ {
				time.Sleep(5 * time.Minute) // интервал проверки
			}
		} else {
			time.Sleep(60 * time.Minute) // интервал проверки
		}

		statusSession, statusServer, public, err := a.statusServSession()
		if err != nil {
			log.Println("[ERROR] Ошибка получения статусов: ", err)
		} else {
			ch, err := checkIfProcessRunning("esme.exe")
			if err != nil {
				log.Println("[ERROR] Ошибка получения списка процессов:", err)
			}
			if !ch || (statusServer == "OFFLINE" && public) { // если сервис не запущен
				var chatMessage string
				time.Sleep(2 * time.Minute)
				_, statusServer, _, err := a.statusServSession()
				if err != nil {
					log.Println("[ERROR] Ошибка получения статусов: ", err)
				} else {
					if statusServer == "OFFLINE" {
						chatMessage = fmt.Sprintf("ВНИМАНИЕ! Станции %s offline\n", a.cfg.hostName) // формируем сообщение
						chatMessage += fmt.Sprintf("Статус сессии - %s\n", statusSession)
						err := a.tg.SendMessage(chatMessage) // отправка сообщения
						if err != nil {
							log.Println("[ERROR] Ошибка отправки сообщения: ", err)
						}
						go a.delayReboot(10)
						log.Printf("[INFO] Станции %s offline\n", a.cfg.hostName) // записываем в лог
						i++                                                       // ведем счет отправленных сообщений
					}
				}
			} else {
				i, y = 0, 0
			}
		}
	}
}

// Проверяет, запущен ли указанный процесс
func checkIfProcessRunning(processName string) (bool, error) {
	cmd := exec.Command("tasklist")
	output, err := cmd.Output()
	if err != nil {
		log.Println("[ERROR] Ошибка получения списка процессов:", err)
		return false, err
	}
	return strings.Contains(string(output), processName), nil
}
