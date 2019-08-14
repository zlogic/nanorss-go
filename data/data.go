package data

import (
	"os"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
)

// DefaultOptions returns the default connection optons for the database, based based on environment variables.
func DefaultOptions() redis.Options {
	addr, ok := os.LookupEnv("REDIS_URL")
	if !ok {
		addr = "localhost:6379"
	}
	return redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
	}
}

// DBService provides configuration and connection objects for accessing Redis.
type DBService struct {
	client *redis.Client
}

// Open opens the database with options and returns a DBService instance.
func Open(options redis.Options) *DBService {
	log.WithField("options", options).Info("Opening client")

	client := redis.NewClient(&options)
	return &DBService{client: client}
}

// Close closes the connection to the database.
func (service *DBService) Close() {
	log.Info("Closing client")
	err := service.client.Close()
	if err != nil {
		log.Fatal(err)
	}
}
