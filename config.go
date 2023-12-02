package main

var ( // true - включение функции, false - выключение
	BotToken          string         // токен бота
	Chat_IDint        int64          // определяем ID чата получателя
	UserID            int64          // ID пользователя, от которого принимаются команды
	serviceChatID     int64          // ID чата для отправки сервисных сообщений
	CommandON         bool   = true  // включить команды управления ботом
	OnlineIpInfo      bool   = true  // инфо по IP online
	CheckAntiCheat    bool   = true  // проверка наличия файлов EasyAntiCheat.exe и EasyAntiCheat_EOS.exe
	CheckFreeSpace    bool   = true  // проверка свободного места на дисках
	AutoUpdateGeolite bool   = false // автообновление файлов GeoLite с Github
	TrialON           bool   = true  // сбор статистики по триальщикам в trial.txt. false - не собирается статистика в trial.txt
	TrialBlock        bool   = false // Блокировка "хитрых" триальщиков. false - нет блокировки
	TrialfileLAN      string = ``    // файл в сети пример `S:\trial.txt`
	StartMessageON    bool   = true  // включение сообщений при начале сессии. false - сообщение не будет приходить
	StopMessageON     bool   = true  // включение о сообщении об окончании сессии. false - сообщение не будет приходить
	shortSessionON    bool   = true  // оповещать о сессиях менее Х минут, выставлять ниже. false - сообщение не будет приходить

)

func getConfigBot() (BotToken string, Chat_IDint, UserID, serviceChatID int64) {
	BotToken = "упвапвапвп" // "enter_your_bot_toket"
	Chat_IDint = 123123     // чат, куда будут приходить информация
	UserID = 123123         // пользователь, от которого будут приниматься команды
	serviceChatID = 0       // чат для сервисных сообщений, 0 - отправка в Chat_IDint
	return BotToken, Chat_IDint, UserID, serviceChatID
}

func minMinute() int {
	minMinuteSession := 10 // выставляем порог отправки сообщений о сессии. значения от 0 до 59
	return minMinuteSession
}
