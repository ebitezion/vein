package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ebitezion/vein/internal/data"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/otel/trace"
)

// application type allows for application dependency injection
type application struct {
	config        config
	log           *log.Logger
	model         data.Models
	startTime     time.Time
	metrics       *metricsStore
	rateLimiter   RateLimiter
	idemStore     IdempotencyStore
	cache         Cache
	queue         JobQueue
	lifecycle     *lifecycle
	plugins       *pluginRegistry
	events        *eventBus
	tracer        trace.Tracer
	infraCleanup  func(context.Context) error
	traceShutdown func(context.Context) error
}

func init() {
	_ = godotenv.Load()
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	db, err := openDB(cfg)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

	logger := log.New(os.Stdout, "[Vein Framework] ", log.Ldate|log.Ltime|log.Lshortfile)

	app, err := newApplication(cfg, logger, data.NewModels(db))
	if err != nil {
		log.Fatal(err)
	}
	app.registerDefaults()

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.lifecycle.Start(ctx); err != nil {
		app.log.Fatalf("failed to start lifecycle hooks: %v", err)
	}

	go func() {
		app.log.Println(" ---------------------------------------------------------------")
		app.log.Printf("  Starting Server on PORT %d and Env as %s", cfg.port, cfg.env)
		app.log.Println(" ---------------------------------------------------------------")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.log.Fatalf("[MAIN|SERVER] %v", err)
		}
	}()

	<-ctx.Done()
	app.log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.lifecycle.Shutdown(shutdownCtx); err != nil {
		app.log.Printf("lifecycle shutdown error: %v", err)
	}
	if app.traceShutdown != nil {
		if err := app.traceShutdown(shutdownCtx); err != nil {
			app.log.Printf("tracing shutdown error: %v", err)
		}
	}
	if app.infraCleanup != nil {
		if err := app.infraCleanup(shutdownCtx); err != nil {
			app.log.Printf("infra shutdown error: %v", err)
		}
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		app.log.Printf("server shutdown error: %v", err)
	}
}

func (app *application) registerDefaults() {
	app.queue.Register("audit.log", func(ctx context.Context, job Job) {
		app.logJSON(map[string]interface{}{
			"event":    "job_processed",
			"job_name": job.Name,
			"payload":  job.Payload,
		})
	})

	app.lifecycle.RegisterOnStart(func(ctx context.Context) error {
		app.queue.Start(ctx, 2)
		app.events.Publish(ctx, "app.started", map[string]string{"app_name": app.config.appName})
		return nil
	})

	app.lifecycle.RegisterOnShutdown(func(ctx context.Context) error {
		app.events.Publish(ctx, "app.stopped", map[string]string{"app_name": app.config.appName})
		return nil
	})
}

func newApplication(cfg config, logger *log.Logger, models data.Models) (*application, error) {
	infra, err := setupInfrastructure(cfg, logger)
	if err != nil {
		return nil, err
	}

	tracer, traceShutdown, err := setupTracing(cfg)
	if err != nil {
		return nil, err
	}

	app := &application{
		config:        cfg,
		log:           logger,
		model:         models,
		startTime:     time.Now().UTC(),
		metrics:       newMetricsStore(),
		rateLimiter:   infra.rateLimiter,
		idemStore:     infra.idempotency,
		cache:         infra.cache,
		queue:         infra.queue,
		lifecycle:     newLifecycle(),
		plugins:       newPluginRegistry(),
		events:        newEventBus(),
		tracer:        tracer,
		infraCleanup:  infra.cleanup,
		traceShutdown: traceShutdown,
	}

	return app, nil
}

// openDB() function returns a sql.DB connection pool.
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
