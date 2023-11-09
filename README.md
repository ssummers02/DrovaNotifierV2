# DrovaNotifierV2
Программа для запуска на станциях в сервисе drova.io, оповещает о начале и окончании сессии через telegram. 

Использовался go1.21.3

Запуск

1. Установите Golang https://go.dev/
2. Создайте нового(или используем старого) бота в Telegram с помощью BotFather https://telegram.me/BotFather
3. Скопируйте все файлы на свой локальный компьютер и распакуйте
4. Если требуется, для удобства переименовываем ПК в винде, так как имя ПК используется для поиска ID станции
5. В файле config.txt заполняем данные по серверам. Имя_ПК = ID станции
6. Также меняем значения authToken, BotToken, Chat_IDint на свои
7. Для компиляции используем copilate и получаем исполняемый файл DrovaNotifierV2.exe
8. Закидываем исполняемый файл и отредактированный config.txt на станцию и запускаем
9. При необходимости добавляем в автозагрузку или планировчщик задач для автостарта
