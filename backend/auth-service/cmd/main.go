package main

import (
	delivery "auth-service/internal/delivery/http"
	"auth-service/internal/migrations"
	"auth-service/internal/repository"
	"auth-service/internal/usecase"
	"auth-service/pkg/config"
	"auth-service/pkg/logger"
	"auth-service/pkg/metrics"
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	_ "auth-service/docs" // Ini akan terisi otomatis setelah menjalankan 'swag init'

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Helpdesk Auth Service API
// @version 1.0
// @description API dokumentasi untuk layanan autentikasi Helpdesk Microservices.
// @host localhost:8081
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	_ = godotenv.Load() // Abaikan error jika .env tidak ada di prod (menggunakan environment variables asli)

	config.Load()

	port := config.AppConfig.AppPort
	jwtSecret := []byte(config.AppConfig.JWTSecret)

	logger.Init("auth-service")

	db, err := repository.NewPostgresDB()
	if err != nil {
		logger.Log.Fatal("failed to connect database:", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Log.Fatal("failed to get database instance:", err)
	}

	if err := migrations.Run(sqlDB, "users"); err != nil {
		logger.Log.Fatal("failed to run migrations:", err)
	}

	repo := repository.NewUserRepository(db)

	if err := bootstrapAdmin(repo); err != nil {
		logger.Log.Error("bootstrap admin: ", err)
	}

	usecase := usecase.NewAuthUsecase(repo)

	// 🔥 TAMBAH refresh repo
	refreshRepo := repository.NewRefreshTokenRepository(db)

	// 🔥 UPDATE constructor
	handler := delivery.NewAuthHandler(usecase, refreshRepo, jwtSecret, db)

	r := gin.Default()

	r.Use(metrics.GinMiddleware())
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Swagger UI hanya di-mount kalau eksplisit diaktifkan (dev only) — jangan
	// expose skema API ke publik secara default.
	if config.AppConfig.EnableSwagger {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// CORS ditangani di API Gateway (satu-satunya entry point publik).
	// auth-service tidak lagi melayani browser secara langsung, jadi tidak
	// perlu header CORS-nya sendiri di sini.

	// 🔥 Optimasi: Trace Middleware
	r.Use(delivery.TraceMiddleware())

	authLimiter := delivery.NewRateLimiter(config.AppConfig.AuthRateLimitRPS, config.AppConfig.AuthRateLimitBurst)
	delivery.RegisterRoutes(r, handler, config.AppConfig.InternalSecret, authLimiter)

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
