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

const exportFilename = "nanorss.json"

func exportData(db *data.DBService) {
	data, err := db.Export()
	if err != nil {
		log.Fatalf("Failed to export json %v", err)
	}
	err = ioutil.WriteFile(exportFilename, []byte(data), 0644)
	if err != nil {
		log.Fatalf("Failed to write file %v", err)
	}
	fmt.Printf("Exported to %v\n", exportFilename)
}

func importData(db *data.DBService) {
	data, err := ioutil.ReadFile(exportFilename)
	if err != nil {
		log.Fatalf("Failed to read file %v", err)
	}
	err = db.Import(string(data))
	if err != nil {
		log.Fatalf("Failed to import data %v", err)
	}
	fmt.Printf("Imported from %v\n", exportFilename)
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
		case "export":
			exportData(db)
		case "import":
			importData(db)
		default:
			log.Fatalf("Unrecognized directive %v", directive)
		}
	}
}
