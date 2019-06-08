package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/handlers"
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
		errs <- http.ListenAndServe(":8080", handlers.CompressHandler(router))
	}()

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errs <- fmt.Errorf("%s", <-c)
	}()

	<-errs
}

const backupFilename = "nanorss.json"

func backupData(db *data.DBService) {
	data, err := db.Backup()
	if err != nil {
		log.WithError(err).Fatal("Failed to back up json")
	}
	err = ioutil.WriteFile(backupFilename, []byte(data), 0644)
	if err != nil {
		log.WithError(err).Fatal("Failed to write file")
	}
	log.WithField("filename", backupFilename).Info("Backed up")
}

func restoreData(db *data.DBService) {
	data, err := ioutil.ReadFile(backupFilename)
	if err != nil {
		log.Fatalf("Failed to read file %v", err)
	}
	err = db.Restore(string(data))
	if err != nil {
		log.Fatalf("Failed to restore data %v", err)
	}
	log.WithField("filename", backupFilename).Info("Restored")
}

func main() {
	// Init data layer
	db, err := data.Open(data.DefaultOptions())
	defer func() {
		db.GC()
		db.Close()
	}()
	if err != nil {
		db.Close()
		log.Fatalf("Failed to open data store %v", err)
	}

	if len(os.Args) < 2 || os.Args[1] == "serve" {
		serve(db)
	} else {
		switch directive := os.Args[1]; directive {
		case "backup":
			backupData(db)
		case "restore":
			restoreData(db)
		default:
			db.Close()
			log.Fatalf("Unrecognized directive %v", directive)
		}
	}
}
