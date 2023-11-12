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
	BotToken, fileConfig, fileGames, hostname      string
	serverID, authToken, mmdbASN, mmdbCity, ipInfo string
	Chat_IDint                                     int64
	isRunning, onlineIpInfo, checkFreeSpace        bool
)

const (
	appName  = "ese.exe"                                            // Имя запускаемого файла
	newTitle = "Drova Notifier v2"                                  // Имя окна программы
	url      = "https://services.drova.io/session-manager/sessions" // инфо по сессиям
)

// для выгрузки названий игр с их ID
type Product struct {
	ProductID string `json:"productId"`
	Title     string `json:"title"`
}

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

// для получения провайдера в оффлайн базе
type ASNRecord struct {
	// AutonomousSystemNumber       uint32 `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
}

// для получения города региона в оффлайн базе
type CityRecord struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	// Country struct {
	// 	Names map[string]string `maxminddb:"names"`
	// } `maxminddb:"country"`
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

// для получения времени запуска windows
type Win32_OperatingSystem struct {
	LastBootUpTime time.Time
}

func main() {
	// Следующие 2 строки вводим свои данные. Чат ID будет с - в начале если это общий чат, и без - если это личка
	BotToken = "11111111:sdsdfsdde" // токен бота
	Chat_IDint = -1002054147798     // определяем ID чата получателя
	onlineIpInfo = false            // false - инфо по IP используя оффлайн базу GeoLite, true - инфо по IP через сайт ipinfo.io
	checkFreeSpace = true

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

	//блок для получения данных из конфига
	_, err = os.Stat(fileConfig)
	if os.IsNotExist(err) {
		// Файл не существует
		log.Printf("Файл %s не существует. %s. %s\n", fileConfig, err, getLine())
	} else {
		bToken, err := readConfig("tokenbot", fileConfig)
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
		onlineIpInf, err := readConfig("onlineIpInfo", fileConfig) // определяем ID чата
		if err != nil {
			log.Printf("Ошибка - %s. %s\n", err, getLine())
		}
		if onlineIpInf == "true" {
			onlineIpInfo = true
		} else if onlineIpInf == "false" {
			onlineIpInfo = false
		}
		checkSpace, err := readConfig("checkFreeSpace", fileConfig) // определяем ID чата
		if err != nil {
			log.Printf("Ошибка - %s. %s\n", err, getLine())
		}
		if checkSpace == "true" {
			checkFreeSpace = true
		} else if checkSpace == "false" {
			checkFreeSpace = false
		}

	}

	mmdbASN = filepath.Join(dir, "GeoLite2-ASN.mmdb")
	mmdbCity = filepath.Join(dir, "GeoLite2-City.mmdb")
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
			go updateRestart(mmdbASN, mmdbCity)
		}
	}

	log.Println(fileGames)
	log.Println(fileConfig)

	gameID(fileGames) // получение списка ID игры - Название игры и сохранение в файл gamesID.txt

	regFolder := `SOFTWARE\ITKey\Esme`
	serverID = regGet(regFolder, "last_server") // получаем ID сервера
	regFolder += `\servers\` + serverID
	authToken = regGet(regFolder, "auth_token") // получаем токен для авторизации

	// Получаем имя ПК
	hostname, err = os.Hostname()
	if err != nil {
		log.Println("Ошибка при получении имени компьютера: ", err, "\nOшибка в строке", getLine())
		return
	}
	messageStartWin(hostname) // проверка времени запуска станции
	if checkFreeSpace {
		diskSpace(hostname) // проверка свободного места на дисках
	}

	for {
		for i := 0; i != 2; { //ждем запуска приложения ese.exe
			time.Sleep(5 * time.Second)                // интервал проверки запущенного процесса
			isRunning = checkIfProcessRunning(appName) // запущено ли приложение
			if isRunning {

				chatMessage := sessionInfo("Start")
				err := SendMessage(BotToken, Chat_IDint, chatMessage)
				if err != nil {
					log.Fatal("Ошибка отправки сообщения: ", err, getLine())
				}
				i = 2 //т.к. приложение запущено, выходим из цикла
			}
		}

		// ждем закрытия процесса ese.exe
		for i := 0; i != 3; {
			isRunning = checkIfProcessRunning(appName)
			if !isRunning {
				time.Sleep(60 * time.Second) // задержка перед получением данных, на случай написания отзыва
				chatMessage := sessionInfo("Stop")
				err := SendMessage(BotToken, Chat_IDint, chatMessage)
				if err != nil {
					log.Fatal("Ошибка отправки сообщения: ", err, getLine())
				}
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

// получаем данные из файла в виде ключ = значение
func readConfig(keys, filename string) (string, error) {
	var gname string
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Ошибка при открытии файла ", filename, ": ", err, getLine())
		fmt.Println("Ошибка при открытии файла ", filename, ": ", err)
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

func SendMessage(botToken string, chatID int64, text string) error {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Println("Ошибка подключения бота: ", err, getLine())
		return err
	}

	message := tgbotapi.NewMessage(chatID, text)

	_, err = bot.Send(message)
	if err != nil {
		log.Println("Ошибка отправки сообщения: ", err, getLine())
		return err
	}

	return nil
}

// получение строки кода где возникла ошибка
func getLine() string {
	_, _, line, _ := runtime.Caller(1)
	lineErr := fmt.Sprintf("\nОшибка в строке: %d", line)
	return lineErr
}

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

func sessionInfo(status string) (infoString string) {
	// Создание HTTP клиента
	client := &http.Client{}

	// Создание нового GET-запроса
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("Failed to create request: ", err, getLine())
	}

	// Установка параметров запроса
	q := req.URL.Query()
	q.Add("server_id", serverID)
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
	var data SessionsData                         // структура SessionsData
	json.Unmarshal([]byte(responseString), &data) // декодируем JSON файл

	comment := strings.ReplaceAll(data.Sessions[0].Abort_comment, ";", ":")
	sessionOn, _ := dateTimeS(data.Sessions[0].Created_on)
	game, _ := readConfig(data.Sessions[0].Product_id, fileGames)
	game = "\nИгра: " + game
	ipInfo = ""
	if onlineIpInfo {
		ipInfo = ipInf(data.Sessions[0].Creator_ip)
	} else {
		ip := net.ParseIP(data.Sessions[0].Creator_ip)
		cityRecord, asnRecord, err := getASNRecord(mmdbCity, mmdbASN, ip)
		if err != nil {
			log.Fatal(err)
		}

		asn := asnRecord.AutonomousSystemOrganization // провайдер клиента
		city := cityRecord.City.Names["ru"]           // город клиента
		region := cityRecord.Subdivision[0].Names["ru"]
		if city != "" {
			ipInfo = "\nГород: " + city
		}
		if region != "" {
			ipInfo += "\nРегион: " + region
		}
		if asn != "" {
			ipInfo += "\nПровайдер: " + asn
		}

	}
	if status == "Start" { // формируем текст для отправки

		billing := data.Sessions[0].Billing_type
		if billing != "" {
			billing = "\nОплата: " + data.Sessions[0].Billing_type
		}
		infoString = "[+]" + hostname + " - " + game + "\n" + data.Sessions[0].Creator_ip + "\n" + sessionOn + ipInfo + billing
		fmt.Println()
		return
	} else { // высчитываем продолжительность сессии и формируем текст для отправки
		_, stopTime := dateTimeS(data.Sessions[0].Finished_on)
		_, startTime := dateTimeS(data.Sessions[0].Created_on)
		sessionDur := "\nПродолжительность сессии: " + dur(stopTime, startTime)
		if comment != "" {
			comment = "\nКомментарий: " + comment
		}
		infoString = "[-]" + hostname + " - " + game + "\n" + data.Sessions[0].Creator_ip + game + sessionDur + comment
		fmt.Println()
		return
	}
}

// конвертирование даты и времени
func dateTimeS(data int64) (string, time.Time) {

	// Создание объекта времени
	seconds := int64(data / 1000)
	nanoseconds := int64((data % 1000) * 1000000)
	t := time.Unix(seconds, nanoseconds)

	// Форматирование времени
	formattedTime := t.Format("02-01-2006 15:04:05")

	return formattedTime, t
}

func dur(stopTime, startTime time.Time) (sessionDur string) {
	duration := stopTime.Sub(startTime).Round(time.Second)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
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
	return
}

// получаем данные из реестра
func regGet(regFolder, keys string) string {

	key, err := registry.OpenKey(registry.LOCAL_MACHINE, regFolder, registry.QUERY_VALUE)
	if err != nil {
		log.Printf("Failed to open registry key: %v\n", err)
	}
	defer key.Close()

	value, _, err := key.GetStringValue(keys)
	if err != nil {
		log.Printf("Failed to read last_server value: %v\n", err)
	}
	// log.Printf("%s - %s: %s\n", keys, regFolder, value)

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
	// region:= cityRecord.City.Names["ru"]
	return &recordCity, &recordASN, err
}

func ipInf(ip string) string {
	apiURL := fmt.Sprintf("https://ipinfo.io/%s/json", ip)

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var ipInfo IPInfoResponse
	err = json.NewDecoder(resp.Body).Decode(&ipInfo)
	if err != nil {
		log.Fatal(err)
	}

	text := "\nГород: " + ipInfo.City + "\nРегион: " + ipInfo.Region + "\nПровайдер: " + ipInfo.ISP
	return text
}

func updateRestart(mmdbASN, mmdbCity string) {
	var previousModTime1, previousModTime2 time.Time
	filePath1 := mmdbASN
	filePath2 := mmdbCity
	// Получаем информацию о файлах
	fileInfo1, err := os.Stat(filePath1)
	if err != nil {
		log.Fatal(err)
	}
	fileInfo2, err := os.Stat(filePath2)
	if err != nil {
		log.Fatal(err)
	}

	// Проверяем время последнего изменения файла
	previousModTime1 = fileInfo1.ModTime()
	previousModTime2 = fileInfo2.ModTime()
	for {
		time.Sleep(60 * time.Second)        // Интервал повторной проверки
		fileInfo1, err = os.Stat(filePath1) // Повторно получаем информацию о файле
		if err != nil {
			log.Fatal(err)
		}
		fileInfo2, err = os.Stat(filePath2)
		if err != nil {
			log.Fatal(err)
		}
		// Проверяем, изменился ли файл по сравнению с предыдущим временем модификации
		if previousModTime1 != fileInfo1.ModTime() || previousModTime2 != fileInfo2.ModTime() {
			log.Println("Файл был изменен. Перезапуск приложения...")
			restart()
		}
	}
}

func restart() {
	// Получаем путь к текущему исполняемому файлу
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	// Запускаем новый экземпляр приложения с помощью os/exec
	cmd := exec.Command(execPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Запускаем новый процесс и не ждем его завершения
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	// Завершаем текущий процесс
	os.Exit(0)
}

func messageStartWin(hostname string) {
	var osInfo []Win32_OperatingSystem
	err := wmi.Query("SELECT LastBootUpTime FROM Win32_OperatingSystem", &osInfo)
	if err != nil {
		log.Fatal(err)
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
			log.Fatal("Ошибка отправки сообщения: ", err, getLine())
		}
	}
}

// проверяем свободное место на дисках
func diskSpace(hostname string) {
	var text string = ""
	partitions, err := disk.Partitions(false)
	if err != nil {
		log.Fatal(err)
	}

	for _, partition := range partitions {
		usageStat, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			log.Printf("Error getting disk usage for %s: %v. %s\n", partition.Mountpoint, err, getLine())
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
			log.Fatal("Ошибка отправки сообщения: ", err, getLine())
		}
	}
}
