package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/zlogic/nanorss-go/data"
	"github.com/zlogic/nanorss-go/fetcher"
	"github.com/zlogic/nanorss-go/server"
	"github.com/zlogic/nanorss-go/worker"
)

func createDefaultUser(db *data.DBService) {
	ch := make(chan *data.User)
	done := make(chan bool)
	haveUsers := false
	go func() {
		for _ = range ch {
			haveUsers = true
		}
		close(done)
	}()
	err := db.ReadAllUsers(ch)
	if err != nil {
		log.Printf("Failed to check users %v", err)
		return
	}
	<-done
	if haveUsers {
		return
	}
	log.Println("Creating default user")
	defaultUser := data.User{}
	defaultUser.SetPassword("default")
	err = db.NewUserService("default").Save(&defaultUser)
	if err != nil {
		log.Printf("Failed to save default user %v", err)
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
	errs := make(chan error, 2)
	router, err := server.CreateRouter(db)
	if err != nil {
		errs <- err
	}

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

func backupData(db *data.DBService) {
	data, err := db.Backup()
	if err != nil {
		log.Fatalf("Failed to back up json %v", err)
	}
	err = ioutil.WriteFile(backupFilename, []byte(data), 0644)
	if err != nil {
		log.Fatalf("Failed to write file %v", err)
	}
	fmt.Printf("Backed up to %v\n", backupFilename)
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
	fmt.Printf("Restored from %v\n", backupFilename)
}

func main() {

	// Init data layer
	db, err := data.Open(data.DefaultOptions())
	defer db.Close()
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
			log.Fatalf("Unrecognized directive %v", directive)
		}
	}
}
