package worker

import (
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

func createTicker() *time.Ticker {
	intervalStr, ok := os.LookupEnv("REFRESH_INTERVAL_MINUTES")
	var interval int
	if !ok {
		intervalStr = "15"
	}
	interval, err := strconv.Atoi(intervalStr)
	if err != nil {
		log.WithField("duration", intervalStr).WithError(err).Error("Cannot parse refresh interval duration")
	}

	return time.NewTicker(time.Duration(interval) * time.Minute)
}

var quit chan struct{}

// Start starts the worker goroutine.
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

// Stop stops the worker goroutine.
func Stop() {
	close(quit)
}
