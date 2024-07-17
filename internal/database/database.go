package database

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/tursodatabase/go-libsql"
)

const DriverName = "libsql"

// OpenSqlxDatabase opens a new sqlx database given the environment variables. If SX_UPLOAD_SQLITE_LOCATION
// is set, it will use that as the sqlite database location. Otherwise, it uses an in-memory database.
func OpenSqlxDatabase() (*sqlx.DB, error) {
	path := ":memory:"
	envPath, found := os.LookupEnv("SX_UPLOAD_SQLITE_LOCATION")
	if found {
		path = "file:" + envPath
	}

	sqlDb, err := sql.Open(DriverName, path)
	if err != nil {
		return nil, err
	}

	db := sqlx.NewDb(sqlDb, DriverName)
	err = createTables(db)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func createTables(db *sqlx.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS
		"files"
		(
			"id" TEXT PRIMARY KEY,
			"user_id" INTEGER NOT NULL,
			"ext" TEXT NOT NULL,
			"blob" BLOB NOT NULL,
			"delete_token" TEXT NOT NULL
		);
    `); err != nil {
		return fmt.Errorf("failed to seed database: %s", err)
	}

	return nil
}
