package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/KenShabby/run_plan_generator/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	connStr := getEnv("DATABASE_URL", "postgres:///run_plan_generator?sslmode=disable")

	sessionSecret := getEnv("SESSION_SECRET", "")
	if sessionSecret == "" {
		log.Fatal("SESSION_SECRET environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("unable to create connection pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("unable to reach database: %v", err)
	}

	fmt.Println("connected to postgres successfully")

	queries := db.New(pool)
	app := newApplication(queries, sessionSecret)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      newServer(app),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("starting server on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
