package main

var ( // true - включение функции, false - выключение
	BotToken          string                  // токен бота
	Chat_IDint        int64                   // определяем ID чата получателя
	UserID            int64                   // ID пользователя, от которого принимаются команды
	CommandON         bool   = true           // включить команды управления ботом
	OnlineIpInfo      bool   = true           // инфо по IP online
	CheckAntiCheat    bool   = true           // проверка наличия файлов EasyAntiCheat.exe и EasyAntiCheat_EOS.exe
	CheckFreeSpace    bool   = true           // проверка свободного места на дисках
	AutoUpdateGeolite bool   = true           // автообновление файлов GeoLite с Github
	TrialBlock        bool   = false          // Блокировка "хитрых" триальщиков
	TrialfileLAN      string = `` // файл в сети пример `S:\trial.txt`
	// Username          string = ""    // авторизация на сетевом хранилище(под будущее расширение функционала)
	// Password          string = ""    // авторизация на сетевом хранилище(под будущее расширение функционала)
	// DiskName          string = ""
	// Share             string = ``
)

func getConfigBot() (BotToken string, Chat_IDint, UserID int64) {
	BotToken = "sdfssdfI" // "enter_your_bot_toket"
	Chat_IDint = 2342342  // чат, куда будут приходить информация
	UserID = 2342342      // пользователь, от которого будут приниматься команды
	return BotToken, Chat_IDint, UserID
}
