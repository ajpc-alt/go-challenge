package database

import (
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func New(user, password, dbname, port string) (db *gorm.DB, close func() error) {
	dsn := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable", user, password, port, dbname)

	// TranslateError maps driver-specific errors onto GORM sentinels, so
	// callers can tell a unique-constraint violation from any other failure.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatalf("failed to connect database: %s", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %s", err)
	}

	return db, sqlDB.Close
}
