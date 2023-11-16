package main

func getConfig() (string, int64, bool) {
	BotToken = "121e1:qew"        // "enter_your_bot_toket"
	Chat_IDint = -234324324234243 // -1234
	// Для отключение блокировки "хитрых" триальщиков меняем значение на false
	trialBlock = false
	return BotToken, Chat_IDint, trialBlock
}
