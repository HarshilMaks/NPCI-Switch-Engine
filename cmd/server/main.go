package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"npci-upi/internal/config"
	"npci-upi/internal/handlers"
	"npci-upi/internal/services"
	"npci-upi/internal/storage"
)

func main() {
	cfg := config.LoadConfig()

	db, err := storage.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := storage.Migrate(ctx, db); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	if err := storage.Seed(ctx, db); err != nil {
		log.Fatalf("failed to seed: %v", err)
	}

	paymentSvc := services.NewPaymentService(db)
	reconciliationSvc := services.NewReconciliationService(db)
	handler := handlers.NewPaymentHandler(paymentSvc, reconciliationSvc)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	handlers.RegisterRoutes(r, handler)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("🚀 Payment Switch Engine starting on %s\n", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
