package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/pauljomy/greenlight/internal/data"
	"github.com/pauljomy/greenlight/internal/env"
	"github.com/pauljomy/greenlight/internal/mailer"
)

const version = "1.0.0"

type dbconfig struct {
	dsn          string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

type limiterConfig struct {
	rps     float64
	burst   int
	enabled bool
}

type SMTPConfig struct {
	host     string
	port     int
	username string
	password string
	sender   string
}

type config struct {
	port    int
	env     string
	db      dbconfig
	limiter limiterConfig
	smtp    SMTPConfig
}

type application struct {
	config config
	logger *slog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
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
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Rate limiter enabled")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "71898b54c3fc7f", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "ba82b361e00868", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP Sender")
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
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	err = app.serve()
	if err != nil {
		logger.Error("Failed to load the server", "error", err)
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
