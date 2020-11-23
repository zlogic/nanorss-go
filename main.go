package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/zlogic/nanorss-go/data"
	"github.com/zlogic/nanorss-go/fetcher"
	"github.com/zlogic/nanorss-go/server"
	"github.com/zlogic/nanorss-go/worker"

	log "github.com/sirupsen/logrus"
)

func createDefaultUser(db *data.DBService) {
	ch := make(chan *data.User)
	done := make(chan bool)
	haveUsers := false
	go func() {
		for range ch {
			haveUsers = true
		}
		close(done)
	}()
	err := db.ReadAllUsers(ch)
	if err != nil {
		log.WithError(err).Error("Failed to check users")
		return
	}
	<-done
	if haveUsers {
		return
	}
	log.Warn("Creating default user")
	defaultUser := data.NewUser("default")
	defaultUser.SetPassword("default")
	err = db.SaveUser(defaultUser)
	if err != nil {
		log.WithError(err).Error("Failed to save default user")
		return
	}
}

func serve(db *data.DBService) {

	// Create default user if necessary
	createDefaultUser(db)

	// Schedule the fetcher worker
	worker.Start(func() {
		fetcher := fetcher.NewFetcher(db)
		fetcher.Refresh()
		db.GC()
	})

	// Create the router and webserver
	services, err := server.CreateServices(db)
	if err != nil {
		log.WithError(err).Error("Error while creating services")
		return
	}
	router, err := server.CreateRouter(services)
	if err != nil {
		log.WithError(err).Error("Error while creating services")
		return
	}

	errs := make(chan error, 2)
	go func() {
		errs <- http.ListenAndServe(":8080", router)
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("%s", <-c)
	}()

	<-errs
}

const backupFilename = "nanorss.json"

func main() {
	// Init data layer
	db, err := data.Open()
	defer func() {
		db.GC()
		db.Close()
	}()
	if err != nil {
		db.Close()
		log.Fatalf("Failed to open data store %v", err)
	}

	serve(db)
}
