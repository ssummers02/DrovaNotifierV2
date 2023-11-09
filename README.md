# DrovaNotifierV2
Программа для запуска на станциях в сервисе drova.io, оповещает о начале и окончании сессии через telegram. 

Использовался go1.21.3

Запуск

1. Установите Golang https://go.dev/
2. Создайте нового(или используем старого) бота в Telegram с помощью BotFather https://telegram.me/BotFather
3. Скопируйте все файлы на свой локальный компьютер и распакуйте
4. Заменяем значения в файле main.go authToken, BotToken, Chat_IDint на свои
5. Если требуется, для удобства переименовываем ПК, так как имя ПК передается в качестве имени станции
6. В файле config.txt заполняем данные по серверам. Имя_ПК = ID станции. В конце файла обязательно оставляем пустую строку
7. Открываем коммандную строку и переходим в распакованную папку
8. Выполняем команду go build -o DrovaNotifierV2.exe main.go. Получаем исполняемый файл DrovaNotifierV2.exe
9. Закидываем исполняемый файл и отредактированный config.txt на станцию и запускаем
10. При необходимости добавляем в автозагрузку или планировчщик задач для автостарта
