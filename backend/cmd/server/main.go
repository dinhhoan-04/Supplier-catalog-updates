package main

import (
	"catalog-sync/internal/config"
	"catalog-sync/internal/handler"
	"catalog-sync/internal/mocksupplier"
	"catalog-sync/internal/repository/postgres"
	"catalog-sync/internal/service"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := config.Load()

	log.Printf("[SERVER] Connecting to database at: %s", cfg.DatabaseURL)
	db, err := postgres.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[FATAL] Database connection failed: %v", err)
	}
	defer db.Close()

	log.Printf("[SERVER] Applying database migrations from: %s", cfg.MigrationPath)
	if err := db.RunMigrations(cfg.MigrationPath); err != nil {
		log.Fatalf("[FATAL] Migration failed: %v", err)
	}
	log.Println("[SERVER] Migrations applied successfully.")

	// Repositories
	productRepo := postgres.NewProductRepository(db)
	jobRepo := postgres.NewJobRepository(db)

	// Clean up stale interrupted jobs from previous application crash
	ctxInit, cancelInit := context.WithTimeout(context.Background(), 5*time.Second)
	interruptedCount, err := jobRepo.MarkStaleRunningJobsAsInterrupted(ctxInit)
	cancelInit()
	if err != nil {
		log.Printf("[WARNING] Could not check stale running jobs: %v", err)
	} else if interruptedCount > 0 {
		log.Printf("[RECOVERY] Recovered from previous crash: marked %d running job(s) as INTERRUPTED. Ready to resume.", interruptedCount)
	}

	// Services & Handlers
	resolver := service.NewConflictResolver()
	supplier := mocksupplier.NewMockSupplier()
	sseHub := handler.NewSSEHub()
	syncEngine := service.NewSyncEngine(db, productRepo, jobRepo, resolver, supplier, sseHub)

	httpHandler := handler.NewHTTPHandler(syncEngine, productRepo, jobRepo, sseHub, db)
	router := handler.NewRouter(httpHandler, sseHub)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second, // Long write timeout for SSE streams
	}

	go func() {
		log.Printf("[SERVER] Catalog Sync Server started on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[SERVER] Received signal %v, shutting down gracefully...", sig)

	// Interrupt any running sync before terminating
	_ = syncEngine.SimulateInterruption()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("[FATAL] Server forced to shutdown: %v", err)
	}

	log.Println("[SERVER] Server exited cleanly.")
}
