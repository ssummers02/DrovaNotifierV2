package main

var (
	BotToken                     string // токен бота
	Chat_IDint                   int64  // определяем ID чата получателя
	UserID                       int64  // ID пользователя, от которого принимаются команды
	commandON                    bool   // включить команды управления ботом
	onlineIpInfo, checkAntiCheat bool
	checkFreeSpace, trialBlock   bool
)

func getConfigBot() (BotToken string, Chat_IDint int64, UserID int64, commandON, trialBlock bool) {
	BotToken = "634234:jkhGJhk" // "enter_your_bot_toket"
	Chat_IDint = 123213         // чат, куда будут приходить информация
	UserID = 123213             // полльзователь, от которого будут приниматься команды
	// включить команды управления ботом. true - включено, false - выключено
	commandON = true
	// Блокировка "хитрых" триальщиков. true - включена, false - выключена
	trialBlock = false

	return BotToken, Chat_IDint, UserID, commandON, trialBlock
}

func getConfig() (onlineIpInfo, checkFreeSpace, checkAntiCheatk bool) {
	// false - инфо по IP используя оффлайн базу GeoLite, true - инфо по IP через сайт ipinfo.io
	onlineIpInfo = false
	// проверка свободного места на дисках. true - проверка включена, false - выключена
	checkFreeSpace = true
	// проверка наличия файлов EasyAntiCheat.exe и EasyAntiCheat_EOS.exe
	checkAntiCheat = true

	return onlineIpInfo, checkFreeSpace, checkAntiCheat
}
