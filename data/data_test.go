package data

import (
	"os"
	"testing"

	"github.com/go-redis/redis"

	"github.com/alicebob/miniredis"
)

var dbService *DBService

func TestMain(m *testing.M) {
	s, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	dbService = Open(redis.Options{Addr: s.Addr()})

	code := m.Run()

	s.Close()
	os.Exit(code)
}

func resetDb() error {
	_, err := dbService.client.FlushAll().Result()
	return err
}
