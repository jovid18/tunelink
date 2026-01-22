package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	DB    *sql.DB
	Redis *redis.Client
}

func Load() *Config {
	db := connectDB()
	rdb := connectRedis()

	return &Config{
		DB:    db,
		Redis: rdb,
	}
}

func connectDB() *sql.DB {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "3306")
	user := getEnv("DB_USER", "tunelink")
	password := getEnv("DB_PASSWORD", "tunelink")
	dbname := getEnv("DB_NAME", "tunelink")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, dbname)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Create table if not exists
	createTable(db)

	log.Println("Connected to MySQL")
	return db
}

func createTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS urls (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		short_url VARCHAR(10) NOT NULL UNIQUE,
		original_url TEXT NOT NULL,
		clicks BIGINT DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_short_url (short_url)
	)`

	if _, err := db.Exec(query); err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

func connectRedis() *redis.Client {
	host := getEnv("REDIS_HOST", "localhost")
	port := getEnv("REDIS_PORT", "6379")

	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port),
	})

	log.Println("Connected to Redis")
	return rdb
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
