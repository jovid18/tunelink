package infrastructure

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	urlDomain "github.com/tunelink/api/internal/domain/url"
)

type Config struct {
	DB      *gorm.DB
	Redis   *redis.Client
	BaseURL string
}

func Load() *Config {
	// Load .env file if exists (ignore error for production)
	_ = godotenv.Load()

	db := connectDB()
	rdb := connectRedis()
	baseURL := mustGetEnv("BASE_URL")

	return &Config{
		DB:      db,
		Redis:   rdb,
		BaseURL: baseURL,
	}
}

func connectDB() *gorm.DB {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "3306")
	user := mustGetEnv("DB_USER")
	password := mustGetEnv("DB_PASSWORD")
	dbname := mustGetEnv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbname)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate
	if err := db.AutoMigrate(&urlDomain.URL{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Connected to MySQL with GORM")
	return db
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

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}
