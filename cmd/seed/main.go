package main

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"

	"github.com/mytheresa/go-hiring-challenge/app/database"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("Error loading .env file: %s", err)
	}

	// Initialize database connection
	db, close := database.New(
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_PORT"),
	)
	defer close()

	dir := os.Getenv("POSTGRES_SQL_DIR")
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("reading directory failed: %v", err)
	}

	// Filter and sort .sql files
	var sqlFiles []os.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, file)
		}
	}
	sort.Slice(sqlFiles, func(i, j int) bool {
		return sqlFiles[i].Name() < sqlFiles[j].Name()
	})

	for _, file := range sqlFiles {
		path := filepath.Join(dir, file.Name())

		content, err := os.ReadFile(path)
		if err != nil {
			log.Printf("reading file %s failed: %v", file.Name(), err)
		}

		sql := string(content)
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("executing %s failed: %v", file.Name(), err)
			return
		}

		log.Printf("Executed %s successfully\n", file.Name())
	}
}
