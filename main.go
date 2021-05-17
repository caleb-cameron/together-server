package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go sigtermCatcher(sigs)

	initConfigs()

	initDB()
	defer closeDB()

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

func sigtermCatcher(sigs chan os.Signal) {
	<-sigs
	log.Println("Shutting down gracefully...")
	closeDB()
	os.Exit(0)
}
