package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pauljomy/greenlight/internal/data"
	"github.com/pauljomy/greenlight/internal/env"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   dbconfig
}

type dbconfig struct {
	dsn          string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type application struct {
	config config
	logger *slog.Logger
	models data.Models
}

func main() {

	godotenv.Load()
	var cfg config

	port := env.GetInt("PORT", 8080)
	environment := env.GetString("ENV", "development")
	greenlightDB := env.GetString("GREENLIGHT_DB_DSN", ``)

	flag.IntVar(&cfg.port, "port", port, "API server port")
	flag.StringVar(&cfg.env, "env", environment, "Environment(development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", greenlightDB, "PostgreSQL DSN")

	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "Postgres max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "Postgres max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "Postgres max idle time")

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB(cfg)
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		os.Exit(1)
	}

	defer db.Close()
	logger.Info("Database connection established")

	app := &application{
		config: cfg,
		logger: logger,
		models: *data.NewModels(db),
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  time.Minute,
	}

	logger.Info("Starting server", "port", cfg.port, "env", cfg.env, "version", version)

	err = srv.ListenAndServe()
	if err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}

}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil

}
