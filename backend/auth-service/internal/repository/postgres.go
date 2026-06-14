package repository

import (
	"auth-service/pkg/config"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgresDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.AppConfig.DBHost,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBName,
		config.AppConfig.DBPort,
	)

	var db *gorm.DB
	var err error

	for i := 0; i < 10; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			log.Println("✅ Connected to database")

			// 🔥 Optimize Connection Pool for Production
			sqlDB, err := db.DB()
			if err != nil {
				return nil, err
			}

			sqlDB.SetMaxOpenConns(100)          // Maksimal koneksi terbuka
			sqlDB.SetMaxIdleConns(10)           // Maksimal koneksi idle
			sqlDB.SetConnMaxLifetime(time.Hour) // Durasi maksimal koneksi bisa digunakan

			return db, nil
		}

		log.Println("⏳ Waiting for database...")
		time.Sleep(2 * time.Second)
	}

	return nil, err
}
