package main

import (
	"log"

	"github.com/shirou/gopsutil/net"
)

func getSpeed(interfaceName string) uint64 {
	// Получаем статистику сетевого интерфейса
	stats, err := net.IOCounters(true)
	if err != nil {
		log.Println("Ошибка при получении статистики сетевого интерфейса:", err)
	}

	// Находим нужный сетевой интерфейс
	var targetInterface net.IOCountersStat
	for _, iface := range stats {
		if iface.Name == interfaceName {
			targetInterface = iface
			break
		}
	}

	// Выводим исходящую скорость
	outputSpeed := targetInterface.BytesSent
	log.Printf("Исходящая скорость на интерфейсе %s: %d байт/сек\n", interfaceName, outputSpeed)
	return outputSpeed
}
