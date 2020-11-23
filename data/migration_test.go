package data

import (
	"database/sql/driver"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigration(t *testing.T) {
	tests := map[string]struct {
		schemaTableExists  bool
		schemaTableError   error
		schemaVersion      int
		schemaVersionError error

		expectCreateSchema bool
		expectError        bool
	}{
		"empty database": {
			schemaTableExists:  false,
			expectCreateSchema: true,
		},
		"version 0": {
			schemaTableExists:  true,
			schemaVersion:      0,
			expectCreateSchema: true,
		},
		"version up to date": {
			schemaTableExists:  true,
			schemaVersion:      1,
			expectCreateSchema: false,
		},
		"cannot test schema table": {
			schemaTableExists:  false,
			schemaTableError:   fmt.Errorf("error"),
			expectCreateSchema: false,
			expectError:        true,
		},
		"cannot get version from schema table": {
			schemaTableExists:  true,
			schemaVersionError: fmt.Errorf("error"),
			expectCreateSchema: false,
			expectError:        true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dbService, mock, err := openMock(true)
			if err != nil {
				t.Fatalf("failed to open mock: %v", err)
			}

			mock.ExpectBegin()

			if !test.schemaTableExists {
				mock.ExpectQuery("SELECT 1 FROM information_schema.tables WHERE table_name=$1").
					WithArgs("schema_version").
					WillReturnRows(mock.NewRows([]string{"one"})).
					WillReturnError(test.schemaTableError)
			} else {
				mock.ExpectQuery("SELECT 1 FROM information_schema.tables WHERE table_name=$1").
					WithArgs("schema_version").
					WillReturnRows(mock.NewRows([]string{"one"}).AddRow("1"))

				mock.ExpectQuery(`SELECT version FROM schema_version`).
					WillReturnRows(mock.NewRows([]string{"version"}).AddRow(test.schemaVersion)).
					WillReturnError(test.schemaVersionError)
			}

			if test.expectCreateSchema {
				mock.ExpectExec(migrationCreateSchema).
					WillReturnResult(driver.ResultNoRows)
			}

			if test.expectError {
				mock.ExpectRollback()
			} else {
				mock.ExpectCommit()
			}
			err = dbService.updateTx(applyMigrations)

			if test.expectError {
				assert.Error(t, err, "database migration should fail")
			} else {
				assert.NoError(t, err, "database migration should not fail")
			}

			err = mock.ExpectationsWereMet()
			assert.NoError(t, err, "check that database expectations were met")
		})
	}
}
