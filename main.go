package main

import (
	"log"
	"time"
)

func main() {
	initConfigs()
	initServer()
	go updateLoop()
	startServer()
}

func updateLoop() {
	log.Println("Starting update loop")
	timer := time.Tick(time.Second / 60.0)

	for {
		select {
		case <-timer:
			update()
		}
	}
}

func update() {
	broadcastGameState()
}
