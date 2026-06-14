package main

import (
	delivery "auth-service/internal/delivery/http"
	"auth-service/internal/domain"
	"auth-service/internal/repository"
	"auth-service/internal/usecase"
	"auth-service/pkg/config"
	"auth-service/pkg/logger"
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	config.Load()

	port := config.AppConfig.AppPort
	jwtSecret := []byte(config.AppConfig.JWTSecret)

	logger.Init("auth-service")

	db, err := repository.NewPostgresDB()
	if err != nil {
		logger.Log.Fatal("failed to connect database:", err)
	}
	sqlDB, _ := db.DB()

	// 🔥 TAMBAH refresh token migration
	db.AutoMigrate(&domain.User{}, &domain.RefreshToken{})

	repo := repository.NewUserRepository(db)
	usecase := usecase.NewAuthUsecase(repo)

	// 🔥 TAMBAH refresh repo
	refreshRepo := repository.NewRefreshTokenRepository(db)

	// 🔥 UPDATE constructor
	handler := delivery.NewAuthHandler(usecase, refreshRepo, jwtSecret, db)

	r := gin.Default()

	// 🔥 Optimasi: CORS Middleware untuk produksi
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Trace-ID")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 🔥 Optimasi: Trace Middleware
	r.Use(delivery.TraceMiddleware())

	delivery.RegisterRoutes(r, handler)

	// 🔥 Implement Graceful Shutdown
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		logger.Log.Info("auth-service running on port " + port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatal("listen: ", err)
		}
	}()

	// Tunggu sinyal interupsi (Ctrl+C atau kill)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Log.Info("Shutting down server...")

	// Berikan waktu 5 detik untuk menyelesaikan request yang sedang berjalan
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Fatal("Server forced to shutdown: ", err)
	}

	// Tutup koneksi DB secara bersih
	sqlDB.Close()
	logger.Log.Info("Server exiting")
}
