package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/StackExchange/wmi"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/oschwald/maxminddb-golang"
	"github.com/shirou/gopsutil/disk"
	"golang.org/x/sys/windows/registry"
)

var (
	fileConfig, fileGames, hostname, ipInfo           string
	serverID, authToken, mmdbASN, mmdbCity, trialfile string
	isRunning                                         bool
)

const (
	appName     = "ese.exe"                                            // Имя запускаемого файла
	newTitle    = "Drova Notifier v2"                                  // Имя окна программы
	UrlSessions = "https://services.drova.io/session-manager/sessions" // инфо по сессиям
	UrlServers  = "https://services.drova.io/server-manager/servers"   // для получения инфо по серверам
	localPort   = "139"                                                // порт для определения IP станции
)

// для выгрузки названий игр с их ID
type Product struct {
	ProductID string `json:"productId"`
	Title     string `json:"title"`
}

// для получения провайдера в оффлайн базе
type ASNRecord struct {
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
}

// для получения города региона в оффлайн базе
type CityRecord struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Subdivision []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
}

// online инфо по IP
type IPInfoResponse struct {
	IP     string `json:"ip"`
	City   string `json:"city"`
	Region string `json:"region"`
	ISP    string `json:"org"`
}

// структура для выгрузки ID и названия серверов
type serverManager []struct {
	Server_id    string `json:"uuid"`
	Name         string `json:"name"`
	User_id      string `json:"user_id"`
	Status       string `json:"state"`
	SessionStart int64  `json:"alive_since"`
}

// для получения времени запуска windows
type Win32_OperatingSystem struct {
	LastBootUpTime time.Time
}

func main() {
	BotToken, Chat_IDint, UserID, trialBlock, commandON = getConfigBot()
	onlineIpInfo, checkFreeSpace, checkAntiCheat = getConfig()

	logFilePath := "log.log" // Имя файла для логирования ошибок
	logFilePath = filepath.Join(filepath.Dir(os.Args[0]), logFilePath)
	// Открываем файл для записи логов
	logFile, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("Ошибка открытия файла", err, getLine())
	}
	defer logFile.Close()
	// Получаем текущую директорию программы
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal("Ошибка получения текущей деректории: ", err, getLine())
	}
	// Устанавливаем файл в качестве вывода для логгера
	log.SetOutput(logFile)

	log.Println("Start program")

	fileGames = filepath.Join(dir, "games.txt")
	fileConfig = filepath.Join(dir, "config.txt")
	trialfile = filepath.Join(dir, "trial.txt")

	// Получаем имя ПК
	hostname, err = os.Hostname()
	if err != nil {
		log.Println("Ошибка при получении имени компьютера: ", err, getLine())
		return
	}

	//блок для получения данных из конфига
	_, err = os.Stat(fileConfig)
	if os.IsNotExist(err) {
		// Файл не существует
		log.Printf("Файл %s не существует. %s\n", fileConfig, err)
	} else {
		bToken, err := readConfig("tokenbot", fileConfig) // определяем токен бота
		if err != nil {
			log.Printf("Ошибка - %s. %s\n", err, getLine())
		}
		if bToken != "" {
			BotToken = bToken // получаем токен этого бота
		}
		Chat_ID, err := readConfig("chatID", fileConfig) // определяем ID чата
		if err != nil {
			log.Printf("Ошибка - %s. %s\n", err, getLine())
		}
		if Chat_ID != "" {
			Chat_IDint, err = strconv.ParseInt(Chat_ID, 10, 64) // конвертируем ID чата в int64
			if err != nil {
				log.Println("Error: ", err, getLine())
			}
		}
		UsrID, err := readConfig("UserID", fileConfig) // определяем ID пользователя
		if err != nil {
			log.Printf("Ошибка - %s. %s\n", err, getLine())
		}
		if UsrID != "" {
			UserID, err = strconv.ParseInt(UsrID, 10, 64) // конвертируем ID чата в int64
			if err != nil {
				log.Println("Error: ", err, getLine())
			}
		}

		onlineIpInfo = takeBoolean("onlineIpInfo")     // настройки получения инфо по IP
		checkFreeSpace = takeBoolean("checkFreeSpace") // проверка свободного места на дисках
		checkAntiCheat = takeBoolean("checkAntiCheat") // проверка папок античитов
		trialBlock = takeBoolean("trialBlock")         // блокировка триальщиков
		commandON = takeBoolean("commandON")
	}

	mmdbASN = filepath.Join(dir, "GeoLite2-ASN.mmdb")   // файл оффлайн базы IP. Провайдер
	mmdbCity = filepath.Join(dir, "GeoLite2-City.mmdb") // файл оффлайн базы IP. Город и область
	_, err = os.Stat(mmdbASN)
	if os.IsNotExist(err) {
		// Файл не существует
		onlineIpInfo = true
		log.Printf("Файл %s не существует. %s. %s\n", fileConfig, err, getLine())
	} else {
		_, err = os.Stat(mmdbCity)
		if os.IsNotExist(err) {
			// Файл не существует
			onlineIpInfo = true
			log.Printf("Файл %s не существует. %s. %s\n", fileConfig, err, getLine())
		} else {
			log.Println(mmdbASN)
			log.Println(mmdbCity)
			go updateGeoLite(mmdbASN, mmdbCity)
		}
	}

	log.Println(fileGames)
	log.Println(fileConfig)

	gameID(fileGames) // получение списка ID игры - Название игры и сохранение в файл gamesID.txt

	regFolder := `SOFTWARE\ITKey\Esme`
	serverID = regGet(regFolder, "last_server") // получаем ID сервера
	regFolder += `\servers\` + serverID
	authToken = regGet(regFolder, "auth_token") // получаем токен для авторизации

	antiCheat(hostname, checkAntiCheat) // проверка античитов
	diskSpace(hostname, checkFreeSpace) // проверка свободного места на дисках
	messageStartWin(hostname)           // проверка времени запуска станции
	if commandON {
		go commandBot(BotToken, hostname, UserID)
	}

	for {
		for i := 0; i != 2; { //ждем запуска приложения ese.exe
			time.Sleep(5 * time.Second)                // интервал проверки запущенного процесса
			isRunning = checkIfProcessRunning(appName) // запущено ли приложение
			if isRunning {
				chatMessage := sessionInfo("Start")
				err := SendMessage(BotToken, Chat_IDint, chatMessage)
				if err != nil {
					log.Println("Ошибка отправки сообщения: ", err, getLine())
				}
				i = 2 //т.к. приложение запущено, выходим из цикла
			}
		}
		// ждем закрытия процесса ese.exe
		for i := 0; i != 3; {
			isRunning = checkIfProcessRunning(appName)
			if !isRunning {
				chatMessage := sessionInfo("Stop")
				err := SendMessage(BotToken, Chat_IDint, chatMessage)
				if err != nil {
					log.Println("Ошибка отправки сообщения: ", err, getLine())
				}

				antiCheat(hostname, checkAntiCheat) // проверка античитов
				diskSpace(hostname, checkFreeSpace) // проверка свободного места на дисках

				i = 3
			}
			time.Sleep(5 * time.Second) // интервал проверки запущенного процесса
		}
	}
}

// Проверяет, запущен ли указанный процесс
func checkIfProcessRunning(processName string) bool {
	cmd := exec.Command("tasklist")
	output, err := cmd.Output()
	if err != nil {
		log.Fatal("Ошибка получения списка процессов:", err, getLine())
	}
	return strings.Contains(string(output), processName)
}

// отправка сообщения ботом
func SendMessage(botToken string, chatID int64, text string) error {
	var i int = 0
	var err error
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Println("Ошибка подключения бота: ", err, getLine())
		return err
	}
	bot.Debug = true

	i = 0
	message := tgbotapi.NewMessage(chatID, text)
	for i = 0; i < 3; i++ {
		_, err = bot.Send(message)
		if err != nil {
			log.Println("Ошибка отправки сообщения: ", err, getLine())
			time.Sleep(1 * time.Second)
			return err
		} else if err == nil {
			i = 3
		}
	}

	return nil
}

// получение строки кода где возникла ошибка
func getLine() string {
	_, _, line, _ := runtime.Caller(1)
	lineErr := fmt.Sprintf("\nОшибка в строке: %d", line)
	return lineErr
}

// получение списка игр с их ID
func gameID(fileGames string) {
	// Отправить GET-запрос на API
	resp, err := http.Get("https://services.drova.io/product-manager/product/listfull2")
	if err != nil {
		fmt.Println("Ошибка при выполнении запроса:", err, getLine())
		return
	}
	defer resp.Body.Close()

	// Прочитать JSON-ответ
	var products []Product
	err = json.NewDecoder(resp.Body).Decode(&products)
	if err != nil {
		fmt.Println("Ошибка при разборе JSON-ответа:", err, getLine())
		return
	}
	// Создать файл для записи
	file, err := os.Create(fileGames)
	if err != nil {
		fmt.Println("Ошибка при создании файла:", err, getLine())
		return
	}
	defer file.Close()

	// Записывать данные в файл
	for _, product := range products {
		line := fmt.Sprintf("%s = %s\n", product.ProductID, product.Title)
		_, err = io.WriteString(file, line)
		if err != nil {
			fmt.Println("Ошибка при записи данных в файл:", err, getLine())
			return
		}
	}
	time.Sleep(1 * time.Second)
}

// конвертирование дат
func dateTimeS(data int64) (string, time.Time) {

	// Создание объекта времени
	seconds := int64(data / 1000)
	nanoseconds := int64((data % 1000) * 1000000)
	t := time.Unix(seconds, nanoseconds)

	// Форматирование времени
	formattedTime := t.Format("02-01-2006 15:04:05")

	return formattedTime, t
}

// высчитываем продолжительность сессии
func dur(stopTime, startTime time.Time) (string, int) {
	var minutes int
	var sessionDur string
	if stopTime.String() != "" {
		duration := stopTime.Sub(startTime).Round(time.Second)
		hours := int(duration.Hours())
		minutes = int(duration.Minutes()) % 60
		seconds := int(duration.Seconds()) % 60
		hou := strconv.Itoa(hours)
		sessionDur = ""
		if hours < 2 {
			sessionDur = sessionDur + "0" + hou + ":"
		} else {
			sessionDur = sessionDur + hou + ":"
		}
		min := strconv.Itoa(minutes)
		if minutes < 10 {
			sessionDur = sessionDur + "0" + min + ":"
		} else {
			sessionDur = sessionDur + min + ":"
		}
		sec := strconv.Itoa(seconds)
		if seconds < 10 {
			sessionDur = sessionDur + "0" + sec
		} else {
			sessionDur = sessionDur + sec
		}
	} else {
		sessionDur = "Ошибка получения времени окончания сессии"
	}
	return sessionDur, minutes
}

// получаем данные из реестра
func regGet(regFolder, keys string) string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, regFolder, registry.QUERY_VALUE)
	if err != nil {
		log.Printf("Failed to open registry key: %v. %s\n", err, getLine())
	}
	defer key.Close()

	value, _, err := key.GetStringValue(keys)
	if err != nil {
		log.Printf("Failed to read last_server value: %v. %s\n", err, getLine())
	}

	return value
}

// offline инфо по IP
func getASNRecord(mmdbCity, mmdbASN string, ip net.IP) (*CityRecord, *ASNRecord, error) {
	dbASN, err := maxminddb.Open(mmdbASN)
	if err != nil {
		return nil, nil, err
	}
	defer dbASN.Close()

	var recordASN ASNRecord
	err = dbASN.Lookup(ip, &recordASN)
	if err != nil {
		return nil, nil, err
	}

	db, err := maxminddb.Open(mmdbCity)
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()

	var recordCity CityRecord
	err = db.Lookup(ip, &recordCity)
	if err != nil {
		return nil, nil, err
	}

	var Subdivision CityRecord
	err = db.Lookup(ip, &Subdivision)
	if err != nil {
		return nil, nil, err
	}
	return &recordCity, &recordASN, err
}

// инфо по IP - ipinfo.io
func onlineDBip(ip string) string {
	apiURL := fmt.Sprintf("https://ipinfo.io/%s/json", ip)
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Println(err, getLine())
	}
	defer resp.Body.Close()

	var ipInfo IPInfoResponse
	err = json.NewDecoder(resp.Body).Decode(&ipInfo)
	if err != nil {
		log.Println(err, getLine())
	}

	text := "\nГород: " + ipInfo.City + "\nРегион: " + ipInfo.Region + "\nПровайдер: " + ipInfo.ISP
	return text
}

// слежение за изменением файлов базы IP
func updateGeoLite(mmdbASN, mmdbCity string) {
	var geoLite = [2]string{mmdbASN, mmdbCity}
	var previousModTime = [2]time.Time{}

	for i := 0; i < len(geoLite); i++ {
		fileInfo, err := os.Stat(geoLite[i])
		if err != nil {
			log.Println(err, getLine())
		}
		// Проверяем время последнего изменения файла
		previousModTime[i] = fileInfo.ModTime()
	}

	for {
		for i := 0; i < len(geoLite); i++ {
			fileInfo, err := os.Stat(geoLite[i])
			if err != nil {
				log.Println(err, getLine())
			}
			if previousModTime[i] != fileInfo.ModTime() {
				log.Println("Файл был изменен. Перезапуск приложения...")
				restart()
			}
		}
		time.Sleep(5 * time.Minute) // Интервал повторной проверки
	}
}

// перезапуск приложения
func restart() {
	// Получаем путь к текущему исполняемому файлу
	execPath, err := os.Executable()
	if err != nil {
		log.Println(err, getLine())
	}

	// Запускаем новый экземпляр приложения с помощью os/exec
	cmd := exec.Command(execPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Запускаем новый процесс и не ждем его завершения
	err = cmd.Start()
	if err != nil {
		log.Println(err, getLine())
	}

	// Завершаем текущий процесс
	os.Exit(0)
}

// оповещение о включении станции
func messageStartWin(hostname string) {
	var osInfo []Win32_OperatingSystem
	err := wmi.Query("SELECT LastBootUpTime FROM Win32_OperatingSystem", &osInfo)
	if err != nil {
		log.Println(err, getLine())
	}

	lastBootUpTime := osInfo[0].LastBootUpTime
	formattedTime := lastBootUpTime.Format("02-01-2006 15:04:05")
	log.Println("Windows запущен - ", formattedTime)
	// Получаем текущее время
	currentTime := time.Now()

	// Вычисляем разницу во времени
	duration := currentTime.Sub(lastBootUpTime)

	// Если прошло менее 5 минут с момента запуска Windows
	if duration.Minutes() < 5 {
		message := fmt.Sprintf("Внимание! Станция %s запущена менее 5 минут назад!\nВремя запуска - %s", hostname, formattedTime)
		err := SendMessage(BotToken, Chat_IDint, message)
		if err != nil {
			log.Println("Ошибка отправки сообщения: ", err, getLine())
		}
	}
}

// проверяем свободное место на дисках
func diskSpace(hostname string, checkFreeSpace bool) {
	if checkFreeSpace {
		var text string = ""
		partitions, err := disk.Partitions(false)
		if err != nil {
			log.Println(err, getLine())
		}

		for _, partition := range partitions {
			usageStat, err := disk.Usage(partition.Mountpoint)
			if err != nil {
				log.Printf("Ошибка получения данных для диска %s: %v. %s\n", partition.Mountpoint, err, getLine())
				continue
			}

			usedSpacePercent := usageStat.UsedPercent

			if usedSpacePercent > 90 {
				text += fmt.Sprintf("На диске %s свободного места менее 10%%\n", partition.Mountpoint)
			}
		}

		// Если text не пустой, значит есть диск со свободным местом менее 10%, отправляем сообщение
		if text != "" {
			message := fmt.Sprintf("Внимание! Станция %s\n%s", hostname, text)
			err := SendMessage(BotToken, Chat_IDint, message)
			if err != nil {
				log.Println("Ошибка отправки сообщения: ", err, getLine())
			}
		}
	}
}

// проверка файлов античитов
func antiCheat(hostname string, checkAntiCheat bool) {
	if checkAntiCheat {
		antiCheat := map[string]string{
			"EasyAntiCheat_EOS": "C:\\Program Files (x86)\\EasyAntiCheat_EOS\\EasyAntiCheat_EOS.exe",
			"EasyAntiCheat":     "C:\\Program Files (x86)\\EasyAntiCheat\\EasyAntiCheat.exe",
		}

		for key, value := range antiCheat {
			filePath := value
			if _, err := os.Stat(filePath); err == nil {
				log.Printf("File %s exists\n", filePath)
			} else if os.IsNotExist(err) {
				log.Printf("Внимание! Станция %s\nОтсутствует файл %s", hostname, key)
				message := fmt.Sprintf("Внимание! Станция %s\nОтсутствует файл %s", hostname, key)
				err := SendMessage(BotToken, Chat_IDint, message)
				if err != nil {
					log.Println("Ошибка отправки сообщения: ", err, getLine())
				}
			} else {
				log.Printf("Ошибка проверки файла %s: %s. %s\n", filePath, err, getLine())
			}
		}
	}
}

// trial - создание или обновление записи по ключу(ip)
func createOrUpdateKeyValue(key string, value int) {
	data := readDataFromFile()
	// Проверяем, существует ли уже ключ в файле
	index := -1
	for i, line := range data {
		if strings.HasPrefix(line, key+"=") {
			index = i
			break
		}
	}
	// Если ключ не существует(-1), добавляем новую запись. Иначе, увеличиваем его значение
	newValue := value
	if index != -1 {
		oldValue, _ := strconv.Atoi(strings.Split(data[index], "=")[1])
		newValue = oldValue + value
		data[index] = key + "=" + strconv.Itoa(newValue)
	} else {
		data = append(data, key+"="+strconv.Itoa(newValue))
	}
	writeDataToFile(data)
}

// trial - получаем значение по ключу(ip)
func getValueByKey(key string) int {
	data := readDataFromFile()
	for _, line := range data {
		parts := strings.Split(line, "=")
		if parts[0] == key {
			value, _ := strconv.Atoi(parts[1])
			return value
		}
	}
	return -1 // Возвращаем -1, если ключ не найден
}

// trial - читаем файл построчно и сздаем слайс
func readDataFromFile() []string {
	file, err := os.Open(trialfile)
	if err != nil {
		return []string{}
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

// trial записываем слайс в файл построчно
func writeDataToFile(data []string) {
	file, err := os.OpenFile(trialfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	for _, line := range data {
		if _, err := file.WriteString(line + "\n"); err != nil {
			panic(err)
		}
	}
}

// получаем данные из файла в виде ключ = значение
func readConfig(keys, filename string) (string, error) {
	var gname string
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Ошибка при открытии файла ", filename, ": ", err, getLine())
		return "Ошибка при открытии файла: ", err
	}
	defer file.Close()

	// Создать сканер для чтения содержимого файла построчно
	scanner := bufio.NewScanner(file)

	// Создать словарь для хранения пары "ключ-значение"
	data := make(map[string]string)

	// Перебирать строки из файла
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, " = ")
		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			data[key] = value
		}
	}

	if value, ok := data[keys]; ok {
		gname = value
	}
	return gname, err
}

// получение данны из файла конфига
func takeBoolean(key string) (value bool) {
	check, err := readConfig(key, fileConfig)
	if err != nil {
		log.Printf("Ошибка - %s. %s\n", err, getLine())
	}
	if check == "true" {
		value = true
	} else if check == "false" {
		value = false

	}
	return
}

// перезагрузка ПК
func rebootPC() {
	cmd := exec.Command("shutdown", "/r", "/t", "0")
	err := cmd.Run()
	if err != nil {
		log.Println(err)
	}
}

func commandBot(tokenBot, hostname string, userID int64) {
	var message, hname string

	hname = strings.ToLower(hostname)
	bot, err := tgbotapi.NewBotAPI(tokenBot)
	if err != nil {
		log.Println(err)
	}

	// таймаут обновления бота
	upd := tgbotapi.NewUpdate(0)
	upd.Timeout = 60

	// получаем обновления от API
	updates := bot.GetUpdatesChan(upd)
	if err != nil {
		log.Println(err)
	}

	for update := range updates {
		//проверяем тип обновления - только новые входящие сообщения
		if update.Message != nil {

			if update.Message.From.ID == userID {
				message = strings.ToLower(update.Message.Text)

				if strings.Contains(message, "/reboot") {
					if strings.Contains(message, hname) { // Проверяем, что в тексте упоминается имя ПК
						log.Println("Перезагрузка ПК по команде из телеграмма")
						message := fmt.Sprintf("Станция %s будет перезагружена по команде из телеграмма", hostname)
						err := SendMessage(BotToken, Chat_IDint, message)
						if err != nil {
							log.Println("Ошибка отправки сообщения: ", err, getLine())
						}
						rebootPC()
					} else {
						messageText := fmt.Sprintf("Имя ПК не совпадает: %s\n", hostname)
						err := SendMessage(BotToken, Chat_IDint, messageText)
						if err != nil {
							log.Println("Ошибка отправки сообщения: ", err, getLine())
						}
					}
				} else if strings.Contains(message, "/status") {
					var serv serverManager                                          // структура serverManager
					json.Unmarshal([]byte(getFromURL(UrlServers, serverID)), &serv) // декодируем JSON файл

					var serverName, status, messageText string

					i := 0
					messageText = fmt.Sprintf("%s\n", hostname)
					for range serv {
						var sessionStart, server_ID string
						serverName = serv[i].Name
						status = serv[i].Status // Получаем статус сервера
						server_ID = serv[i].Server_id

						if status == "BUSY" || status == "HANDSHAKE" { // Получаем время начала, если станция занят
							var data SessionsData                                             // структура SessionsData
							json.Unmarshal([]byte(getFromURL(UrlSessions, server_ID)), &data) // декодируем JSON файл
							startTime, _ := dateTimeS(data.Sessions[0].Created_on)
							sessionStart = fmt.Sprintf("\n%s", startTime)
						} else {
							sessionStart = ""
						}
						messageText += fmt.Sprintf("%s - %s%s\n", serverName, status, sessionStart)
						i++
					}

					err := SendMessage(BotToken, Chat_IDint, messageText)
					if err != nil {
						log.Println("Ошибка отправки сообщения: ", err, getLine())
					}
				} else {
					messageText := "Неизвестная команда"
					err := SendMessage(BotToken, Chat_IDint, messageText)
					if err != nil {
						log.Println("Ошибка отправки сообщения: ", err, getLine())
					}
				}
			}
			log.Printf("Сообщение от %d: %s", update.Message.From.ID, update.Message.Text)
		}
	}
}

func getFromURL(url, server_ID string) string {
	// Создание HTTP клиента
	client := &http.Client{}

	// Создание нового GET-запроса
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Failed to create request: ", err, getLine())
	}

	// Установка параметров запроса
	q := req.URL.Query()
	q.Add("server_id", server_ID)
	req.URL.RawQuery = q.Encode()

	// Установка заголовка X-Auth-Token
	req.Header.Set("X-Auth-Token", authToken)

	// Отправка запроса и получение ответа
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Failed to send request: ", err, getLine())
	}
	defer resp.Body.Close()

	// Запись ответа в строку
	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		log.Fatal("Failed to write response to buffer: ", err, getLine())
	}

	responseString := buf.String()
	return responseString
}

func getInterface() (localAddr, nameInterface string) {
	var maxSpeed uint64 = 0
	var localIP string
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Printf("Ошибка получения интерфейсов. %s. %s\n", err, getLine())
	}
	maxSpeed = 0
	for _, interf := range interfaces {
		addrs, err := interf.Addrs()
		nameI := interf.Name
		if err != nil {
			log.Printf("Ошибка получения ip адресов. %s. %s\n", err, getLine())
		}
		for _, add := range addrs {
			if ip, ok := add.(*net.IPNet); ok {
				log.Println("адрес = ", ip)
				localIP = ip.String()
			}
		}
		speed := getSpeed(nameI)
		if speed > maxSpeed {
			maxSpeed = speed
			localAddr = localIP
			nameInterface = nameI
		}
	}
	return localAddr, nameInterface
}

func offlineDBip(ip string) string {
	var city, region, asn string = "", "", ""

	cityRecord, asnRecord, err := getASNRecord(mmdbCity, mmdbASN, net.ParseIP(ip))
	if err != nil {
		log.Println(err)
	}

	asn = asnRecord.AutonomousSystemOrganization // провайдер клиента
	if err != nil {
		log.Println(err, getLine())
		asn = ""
	}

	if val, ok := cityRecord.City.Names["ru"]; ok { // город клиента
		city = val
		if err != nil {
			log.Println(err, getLine())
			city = ""
		}
	} else {
		if val, ok := cityRecord.City.Names["en"]; ok {
			city = val
			if err != nil {
				log.Println(err, getLine())
				city = ""
			}
		}
	}

	if len(cityRecord.Subdivision) > 0 {
		if val, ok := cityRecord.Subdivision[0].Names["ru"]; ok { // регион клиента
			region = val
			if err != nil {
				log.Println(err, getLine())
				region = ""
			}
		} else {
			if val, ok := cityRecord.Subdivision[0].Names["en"]; ok {
				region = val
				if err != nil {
					log.Println(err, getLine())
					region = ""
				}
			}
		}
	}

	if city != "" {
		ipInfo = " - " + city
	}
	if region != "" {
		ipInfo += " - " + region
	}
	if asn != "" {
		ipInfo += " - " + asn
	}
	return ipInfo
}
