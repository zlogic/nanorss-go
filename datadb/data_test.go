package datadb

import (
	"os"
	"testing"

	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {
	tests := map[string]struct {
		useRealDatabase bool
		databaseURL     string
		expectError     bool
	}{
		"working URL": {
			useRealDatabase: true,
			expectError:     false,
		},
		"empty URL": {
			databaseURL: "",
			expectError: true,
		},
		"invalid URL": {
			databaseURL: "postgres://*",
			expectError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.useRealDatabase {
				useRealDatabase()
			} else {
				os.Setenv("DATABASE_URL", test.databaseURL)
			}

			dbService, err := Open()
			if test.expectError {
				assert.Error(t, err, "database connection should fail")
				assert.Nil(t, dbService)
			} else {
				assert.NoError(t, err, "database connection should not fail")
				assert.NotNil(t, dbService)

				dbService.Close()
			}
		})
	}
}

// useRealDatabase switches to use the real database specified in TEST_DATABASE_URL.
func useRealDatabase() {
	url, _ := os.LookupEnv("TEST_DATABASE_URL")
	os.Setenv("DATABASE_URL", url)
}
