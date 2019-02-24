package worker

import (
	"log"
	"os"
	"strconv"
	"time"
)

func createTicker() *time.Ticker {
	intervalStr, ok := os.LookupEnv("REFRESH_INTERVAL_MINUTES")
	var interval int
	if !ok {
		intervalStr = "15"
	}
	interval, err := strconv.Atoi(intervalStr)
	if err != nil {
		log.Printf("Cannot parse refresh interval duration %v %v", intervalStr, interval)
	}

	return time.NewTicker(time.Duration(interval) * time.Minute)
}

var quit chan struct{}

func Start(task func()) {
	ticker := createTicker()
	quit = make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				task()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func Stop() {
	close(quit)
}
