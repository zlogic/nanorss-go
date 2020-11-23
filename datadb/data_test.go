package datadb

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v4/stdlib"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	log "github.com/sirupsen/logrus"
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
				err := cleanRealDatabase()
				assert.NoError(t, err, "database cleanup should not fail")
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

// cleanRealDatabase drops all tables in the real database specified in TEST_DATABASE_URL.
func cleanRealDatabase() error {
	url, _ := os.LookupEnv("TEST_DATABASE_URL")
	db, err := sql.Open("pgx", url)
	if err != nil {
		return err
	}
	defer db.Close()

	tables := make([]string, 0)

	err = tx(db, func(tx *sql.Tx) error {
		tableRows, err := tx.Query(`SELECT tablename FROM pg_tables WHERE schemaname = 'public';`)
		if err != nil {
			return err
		}
		for tableRows.Next() {
			var name string
			if err := tableRows.Scan(&name); err != nil {
				return err
			}
			tables = append(tables, name)
		}
		return nil
	}, true)
	if err != nil {
		return err
	}
	if len(tables) == 0 {
		return nil
	}

	err = tx(db, func(tx *sql.Tx) error {
		_, err = tx.Exec(fmt.Sprintf("DROP TABLE %s;", strings.Join(tables, ",")))
		if err != nil {
			return err
		}
		return nil
	}, false)
	if err != nil {
		return err
	}

	return nil
}

// openMock returns a mock DBService.
func openMock(exactMatch bool) (*DBService, sqlmock.Sqlmock, error) {
	queryMatcher := sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp)
	if exactMatch {
		queryMatcher = sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual)
	}
	db, mock, err := sqlmock.New(queryMatcher)
	if err != nil {
		return nil, nil, err
	}
	return &DBService{db: db}, mock, nil
}

var dbService *DBService

func TestMain(m *testing.M) {
	var err error
	useRealDatabase()
	dbService, err = Open()
	if err != nil {
		log.WithError(err).Fatal("failed to initialize database")
	}
	code := m.Run()
	dbService.Close()
	os.Exit(code)
}
