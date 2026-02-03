package infrastructure

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	urlDomain "github.com/tunelink/api/internal/domain/url"
)

type Config struct {
	DB           *gorm.DB
	Redis        *redis.Client
	RedisEnabled bool
	BaseURL      string
}

func Load() *Config {
	// Load .env file if exists (ignore error for production)
	_ = godotenv.Load()

	db := connectDB()
	rdb, redisEnabled := connectRedis()
	baseURL := mustGetEnv("BASE_URL")

	return &Config{
		DB:           db,
		Redis:        rdb,
		RedisEnabled: redisEnabled,
		BaseURL:      baseURL,
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

	// Connection pool 설정
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}
	sqlDB.SetMaxOpenConns(25)               // Pod당 최대 25개 연결
	sqlDB.SetMaxIdleConns(10)               // 유휴 연결 10개 유지
	sqlDB.SetConnMaxLifetime(5 * time.Minute) // 연결 수명 5분

	// Auto migrate
	if err := db.AutoMigrate(&urlDomain.URL{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Connected to MySQL with GORM (pool: max=25, idle=10)")
	return db
}

func connectRedis() (*redis.Client, bool) {
	host := getEnv("REDIS_HOST", "")
	if host == "" {
		log.Println("REDIS_HOST not set, Redis disabled")
		return nil, false
	}

	port := getEnv("REDIS_PORT", "6379")
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port),
	})

	// Verify connection with ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("Redis connection failed: %v, Redis disabled", err)
		return nil, false
	}

	log.Println("Connected to Redis")
	return rdb, true
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
