package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"npci-upi/internal/config"
	"npci-upi/internal/handlers"
	"npci-upi/internal/services"
	"npci-upi/internal/storage"
)

func main() {
	cfg := config.Load()

	db, err := storage.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := storage.Migrate(db); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	if err := storage.Seed(db, cfg); err != nil {
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


