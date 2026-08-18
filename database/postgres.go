package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB is the global database connection pool.
var DB *sql.DB

// Connect initializes the PostgreSQL connection pool.
func Connect(dsn string) error {
	var err error
	DB, err = sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("error opening database: %w", err)
	}

	// Configure connection pool
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(5 * time.Minute)
	DB.SetConnMaxIdleTime(2 * time.Minute)

	// Verify connection
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("error connecting to database: %w", err)
	}

	log.Println("✅ Connected to PostgreSQL database")
	return nil
}

// Close gracefully closes the database connection pool.
func Close() {
	if DB != nil {
		if err := DB.Close(); err != nil {
			log.Printf("⚠️  Error closing database: %v", err)
		} else {
			log.Println("🔒 Database connection closed")
		}
	}
}

// Health checks if the database connection is alive.
func Health() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}
	return DB.Ping()
}

// Rebind replaces all "?" placeholders in a query with "$1", "$2", etc.
func Rebind(query string) string {
	parts := strings.Split(query, "?")
	if len(parts) == 1 {
		return query
	}
	var sb strings.Builder
	for i := 0; i < len(parts)-1; i++ {
		sb.WriteString(parts[i])
		sb.WriteString(fmt.Sprintf("$%d", i+1))
	}
	sb.WriteString(parts[len(parts)-1])
	return sb.String()
}
