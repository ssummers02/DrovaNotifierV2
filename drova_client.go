package main

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type DrovaClient struct {
	http.Client
}

func NewDrovaClient() *DrovaClient {
	client := http.Client{}
	client.Timeout = 10 * time.Second

	return &DrovaClient{Client: client}
}

func (c *DrovaClient) GetGame(dir string) error {
	fileGames := filepath.Join(dir, "games.txt")

	// Отправить GET-запрос на API
	respGame, err := http.Get("https://services.drova.io/product-manager/product/listfull2")
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("[ERROR] respGame.Body.Close(): ", err)
		}
	}(respGame.Body)

	// Прочитать JSON-ответ
	var products []Product
	err = json.NewDecoder(respGame.Body).Decode(&products)
	if err != nil {
		return fmt.Errorf("ошибка при декодировании JSON-ответа: %v", err)
	}
	// Создать файл для записи
	file, err := os.Create(fileGames)
	if err != nil {
		return fmt.Errorf("ошибка при создании файла: %v", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Println("[ERROR] fileGames.Close(): ", err)
		}
	}(file)

	builder := strings.Builder{}
	// Записывать данные в файл
	for _, product := range products {
		builder.WriteString(fmt.Sprintf("%s = %s\n", product.ProductID, product.Title))
	}
	_, err = io.WriteString(file, builder.String())
	if err != nil {
		return fmt.Errorf("ошибка при записи данных в файл: %v", err)
	}

	return nil
}
