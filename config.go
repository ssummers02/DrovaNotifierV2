package main

type Config struct {
	BotToken          string  `yaml:"bot_token" yaconf:"required"`                            // Токен бота
	ChatID            int64   `yaml:"chat_id" yaconf:"required"`                              // ID чата, куда будут приходить сообщения
	UserID            int64   `yaml:"user_id" yaconf:"required"`                              // ID пользователя, от которого будут приниматься команды
	ServiceChatID     int64   `yaml:"service_chat_id" yaconf:"required"`                      // ID чата, куда будут приходить сервисные сообщения
	CommandON         bool    `yaml:"command_on" yaconf:"required,default=true"`              // управление ботом через чат ТГ
	viewHostname      bool    `yaml:"view_hostname" yaconf:"required,default=true"`           // вывод имени ПК в сообщениях
	oneBot4all        bool    `yaml:"one_bot_4all" yaconf:"required,default=true"`            // Один бот для всех станций
	OnlineIpInfo      bool    `yaml:"online_ip_info" yaconf:"required,default=false"`         // Получение инфо по IP
	AutoUpdateGeolite bool    `yaml:"auto_update_geolite" yaconf:"required,default=false"`    // Автообновление базы GeoLite2-City.mmdb
	CheckAntiCheat    bool    `yaml:"check_anticheat" yaconf:"required,default=true"`         // проверка папок античитов
	CheckFreeSpace    bool    `yaml:"check_free_space" yaconf:"required,default=true"`        // Проверка свободного места на дисках
	CheckTempON       bool    `yaml:"check_temp_on" yaconf:"required,default=true"`           // Проверка температуры
	FANt              float64 `yaml:"fan_t" yaconf:"required,default=75"`                     // Порог проверки работы вентиляторов видеокарты
	FANrpm            float64 `yaml:"fan_rpm" yaconf:"required,default=800"`                  // Минимальные обороты при FANt
	CPUtmax           float64 `yaml:"cpu_tmax" yaconf:"required,default=85"`                  // Порог температуры процессора
	GPUtmax           float64 `yaml:"gpu_tmax" yaconf:"required,default=85"`                  // Порог температуры ядра видеокарты
	GPUhsTmax         float64 `yaml:"gpu_hs_tmax" yaconf:"required,default=90"`               // Порог температуры HotSpot видеокарты
	DeltaT            float64 `yaml:"delta_t" yaconf:"required,default=5"`                    // Дельта температуры
	TrialON           bool    `yaml:"trial_on" yaconf:"required,default=false"`               // Вести статистику триала
	TrialBlock        bool    `yaml:"trial_block" yaconf:"required,default=false"`            // Блокировка триальщиков
	TrialfileLAN      string  `yaml:"trial_file_lan" yaconf:"required,default=trial.txt"`     // Файл с триальщиками
	StartMessageON    bool    `yaml:"start_message_on" yaconf:"required,default=true"`        // Включение о сообщении о начале сессии. false - сообщение не будет приходить
	StopMessageON     bool    `yaml:"stop_message_on" yaconf:"required,default=true"`         // Включение о сообщении об окончании сессии. false - сообщение не будет приходить
	ShortSessionON    bool    `yaml:"short_session_on" yaconf:"required,default=true"`        // Короткие сообщения
	minMinute         int     `yaml:"min_minute" yaconf:"required,default=10"`                // Минимальное время сессии в минутах
	CommentMessageON  bool    `yaml:"comment_message_on" yaconf:"required,default=true"`      // Включить комментарии
	mmdbASN           string  `yaml:"mmdb_asn" yaconf:"required,default=GeoLite2-ASN.mmdb"`   // Файл офлайн базы IP. Провайдер
	mmdbCity          string  `yaml:"mmdb_city" yaconf:"required,default=GeoLite2-City.mmdb"` // Файл офлайн базы IP. Город и область
	hostName          string
	serverID          string
	authToken         string
}
