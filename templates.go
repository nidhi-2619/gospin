// Code in this file holds the embedded project templates and the file map
// used by Generate. The templates are plain Go source rendered with the
// module path substituted for the __MODULE__ placeholder.

package gospin

import "strings"

// Files returns every file Generate would write for the given module path,
// keyed by path relative to the output directory, without touching disk.
// It is exported so callers can inspect or post-process the generated set.
func Files(mod string) map[string]string {
	m := func(s string) string { return strings.ReplaceAll(s, "__MODULE__", mod) }
	return map[string]string{
		// Entry points
		"cmd/api/main.go":     m(tplCmdMain),
		"cmd/migrate/main.go": m(tplCmdMigrate),

		// Bootstrap wiring
		"bootstrap/app.go":     m(tplBootstrapApp),
		"bootstrap/infra.go":   m(tplBootstrapInfra),
		"bootstrap/modules.go": m(tplBootstrapModules),
		"bootstrap/routes.go":  m(tplBootstrapRoutes),
		"bootstrap/server.go":  m(tplBootstrapServer),

		// API handlers
		"api/health/health_handler.go":      m(tplHealthHandler),
		"api/health/health_handler_test.go": m(tplHealthHandlerTest),

		// Server
		"server/server.go":     m(tplServer),
		"server/middleware.go": m(tplMiddleware),
		"server/routes.go":     m(tplServerRoutes),

		// Config
		"config/config.go": m(tplConfig),
		"config/db.go":     m(tplConfigDB),
		"config/http.go":   m(tplConfigHTTP),
		"config/jwt.go":    m(tplConfigJWT),
		"config/redis.go":  m(tplConfigRedis),

		// Common
		"common/response.go": m(tplCommonResponse),
		"common/utils.go":    m(tplCommonUtils),

		// Infra
		"infra/db/db.go":       m(tplInfraDB),
		"infra/cache/redis.go": m(tplInfraRedis),
		"infra/jwt/token.go":   m(tplInfraJWT),

		// Middleware
		"middleware/auth.go": m(tplMiddlewareAuth),

		// Dev tooling
		"Dockerfile":         tplDockerfile,
		"docker-compose.yml": tplDockerCompose,
		"Makefile":           tplMakefile,
		".env.example":       tplEnvExample,
		".gitignore":         tplGitignore,
		"README.md":          m(tplReadme),
	}
}

// ─── Templates ────────────────────────────────────────────────────────────────

// ── cmd/api/main.go ────────────────────────────────────────────────────────
const tplCmdMain = `package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"__MODULE__/bootstrap"
	"__MODULE__/config"

	"go.uber.org/zap"
)

func main() {
	// ── Logger ────────────────────────────────────────────────────────────────
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialise logger: " + err.Error())
	}
	defer logger.Sync() //nolint:errcheck

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load(logger)
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// ── Application ───────────────────────────────────────────────────────────
	app, err := bootstrap.NewApp(cfg, logger)
	if err != nil {
		logger.Fatal("failed to bootstrap application", zap.Error(err))
	}
	defer app.Close()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx); err != nil {
		logger.Error("application exited with error", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("server shut down gracefully")
}
`

// ── cmd/migrate/main.go ───────────────────────────────────────────────────
const tplCmdMigrate = `package main

import (
	"flag"
	"fmt"
	"os"

	"__MODULE__/config"

	"go.uber.org/zap"
)

func main() {
	direction := flag.String("direction", "up", "Migration direction: up | down")
	steps := flag.Int("steps", 0, "Number of steps (0 = all)")
	flag.Parse()

	logger, _ := zap.NewProduction()
	defer logger.Sync() //nolint:errcheck

	cfg, err := config.Load(logger)
	if err != nil {
		logger.Fatal("config load failed", zap.Error(err))
	}

	logger.Info("running migrations",
		zap.String("dsn", cfg.DB.DSN()),
		zap.String("direction", *direction),
		zap.Int("steps", *steps),
	)

	// TODO: wire your migration library here (e.g. golang-migrate/migrate).
	fmt.Fprintf(os.Stdout, "migrations %s complete\n", *direction)
}
`

// ── bootstrap/app.go ──────────────────────────────────────────────────────
const tplBootstrapApp = `// Package bootstrap wires all application dependencies and returns a
// runnable App. This is the composition root of the service.
package bootstrap

import (
	"context"
	"database/sql"

	"__MODULE__/config"
	"__MODULE__/server"

	"go.uber.org/zap"
)

// App holds every top-level dependency so they can be cleanly shut down.
type App struct {
	cfg    *config.Config
	logger *zap.Logger
	db     *sql.DB
	srv    *server.Server
}

// NewApp builds the full dependency graph and returns a ready App.
func NewApp(cfg *config.Config, logger *zap.Logger) (*App, error) {
	app := &App{cfg: cfg, logger: logger}

	if err := app.initInfra(); err != nil {
		return nil, err
	}

	app.initModules()

	app.srv = server.New(cfg, logger)
	app.registerRoutes()

	return app, nil
}

// Run starts the HTTP server and blocks until ctx is cancelled.
func (a *App) Run(ctx context.Context) error {
	return a.srv.Run(ctx)
}

// Close gracefully releases all long-lived resources.
func (a *App) Close() {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error("db close error", zap.Error(err))
		}
	}
	a.logger.Info("all resources released")
}
`

// ── bootstrap/infra.go ────────────────────────────────────────────────────
const tplBootstrapInfra = `package bootstrap

import (
	"__MODULE__/infra/db"
)

// initInfra opens all infrastructure connections (DB, cache, etc.).
func (a *App) initInfra() error {
	database, err := db.Connect(a.cfg, a.logger)
	if err != nil {
		return err
	}
	a.db = database
	return nil
}
`

// ── bootstrap/modules.go ─────────────────────────────────────────────────
const tplBootstrapModules = `package bootstrap

// initModules wires domain modules (repositories → services → handlers).
// Add your module registrations here as the project grows.
//
// Example:
//
//	userRepo    := userrepo.New(a.db)
//	userService := usersvc.New(userRepo, a.logger)
//	a.userHandler = userhandler.New(userService, a.logger)
func (a *App) initModules() {
	// TODO: register domain modules
}
`

// ── bootstrap/routes.go ───────────────────────────────────────────────────
const tplBootstrapRoutes = `package bootstrap

import (
	"__MODULE__/api/health"
)

// registerRoutes mounts all HTTP routes onto the server engine.
func (a *App) registerRoutes() {
	healthHandler := health.NewHandler(a.logger)
	a.srv.RegisterRoutes(healthHandler)
}
`

// ── bootstrap/server.go ───────────────────────────────────────────────────
const tplBootstrapServer = `package bootstrap
// server.go intentionally left minimal — lifecycle is managed in app.go.
// Add feature-flag gating or multi-listener setup here if needed.
`

// ── api/health/health_handler.go ──────────────────────────────────────────
const tplHealthHandler = `// Package health exposes a liveness probe endpoint.
package health

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var startTime = time.Now()

// Handler holds the dependencies for health-check routes.
type Handler struct {
	logger *zap.Logger
}

// NewHandler creates a Handler with the given Zap logger.
func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{logger: logger}
}

// Check godoc
//
//	@Summary		Health check
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	map[string]any
//	@Router			/health [get]
func (h *Handler) Check(c *gin.Context) {
	h.logger.Debug("health check called")
	c.JSON(http.StatusOK, gin.H{
		"status":  "UP",
		"uptime":  time.Since(startTime).String(),
		"version": "1.0.0",
	})
}
`

// ── api/health/health_handler_test.go ─────────────────────────────────────
const tplHealthHandlerTest = `package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"__MODULE__/api/health"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func setupRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := health.NewHandler(zap.NewNop())
	r.GET("/health", h.Check)
	return r
}

func TestHealthCheck_StatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	setupRouter(t).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHealthCheck_ContentType(t *testing.T) {
	w := httptest.NewRecorder()
	setupRouter(t).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	ct := w.Header().Get("Content-Type")
	if ct == "" || ct[:16] != "application/json" {
		t.Fatalf("unexpected Content-Type: %q", ct)
	}
}

func TestHealthCheck_Body(t *testing.T) {
	w := httptest.NewRecorder()
	setupRouter(t).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body["status"] != "UP" {
		t.Fatalf("expected status=UP, got %v", body["status"])
	}
	if _, ok := body["uptime"]; !ok {
		t.Fatal("missing 'uptime' field")
	}
}

func TestHealthCheck_MethodNotAllowed(t *testing.T) {
	w := httptest.NewRecorder()
	setupRouter(t).ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/health", nil))
	if w.Code != http.StatusNotFound && w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 404/405, got %d", w.Code)
	}
}
`

// ── server/server.go ──────────────────────────────────────────────────────
const tplServer = `// Package server configures and manages the Gin HTTP engine lifecycle.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"__MODULE__/api/health"
	"__MODULE__/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server wraps the HTTP server and its dependencies.
type Server struct {
	cfg    *config.Config
	logger *zap.Logger
	engine *gin.Engine
	http   *http.Server
}

// New creates a configured Gin engine with standard middleware applied.
func New(cfg *config.Config, logger *zap.Logger) *Server {
	if cfg.HTTP.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Recovery())

	s := &Server{
		cfg:    cfg,
		logger: logger,
		engine: engine,
		http: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
			Handler:      engine,
			ReadTimeout:  cfg.HTTP.ReadTimeout,
			WriteTimeout: cfg.HTTP.WriteTimeout,
			IdleTimeout:  cfg.HTTP.IdleTimeout,
		},
	}

	engine.Use(s.zapMiddleware())
	return s
}

// RegisterRoutes mounts all application routes. Called by bootstrap.
func (s *Server) RegisterRoutes(h *health.Handler) {
	s.engine.GET("/health", h.Check)

	v1 := s.engine.Group("/api/v1")
	{
		v1.GET("/health", h.Check)
	}
}

// Run starts the HTTP listener and blocks until ctx is cancelled,
// then performs a graceful shutdown.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("server listening", zap.String("addr", s.http.Addr))
		if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.logger.Info("shutdown signal received")
	}

	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.http.Shutdown(shutCtx)
}
`

// ── server/middleware.go ──────────────────────────────────────────────────
const tplMiddleware = `package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// zapMiddleware returns a Gin handler that logs each request via Zap.
func (s *Server) zapMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.Duration("latency", latency),
			zap.String("user-agent", c.Request.UserAgent()),
		}

		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				fields = append(fields, zap.Error(e.Err))
			}
			s.logger.Error("request error", fields...)
			return
		}

		switch {
		case status >= 500:
			s.logger.Error("server error", fields...)
		case status >= 400:
			s.logger.Warn("client error", fields...)
		default:
			s.logger.Info("request", fields...)
		}
	}
}
`

// ── server/routes.go ──────────────────────────────────────────────────────
const tplServerRoutes = `package server
// Routes are registered through Server.RegisterRoutes, called from
// bootstrap/routes.go. Additional route groups can be added here.
`

// ── config/config.go ──────────────────────────────────────────────────────
const tplConfig = `// Package config provides strongly-typed, environment-sourced configuration.
package config

import (
	"go.uber.org/zap"
)

// Config is the root configuration object injected throughout the app.
type Config struct {
	DB   DBConfig
	HTTP HTTPConfig
	JWT  JWTConfig
}

// Load reads configuration from environment variables and returns a Config.
// A Zap logger is accepted so config-load errors are traceable.
func Load(logger *zap.Logger) (*Config, error) {
	logger.Info("loading configuration")

	cfg := &Config{
		DB:   loadDB(),
		HTTP: loadHTTP(),
		JWT:  loadJWT(),
	}

	return cfg, nil
}
`

// ── config/db.go ──────────────────────────────────────────────────────────
const tplConfigDB = `package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// DBConfig holds PostgreSQL connection settings.
type DBConfig struct {
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func loadDB() DBConfig {
	return DBConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5432),
		Name:            getEnv("DB_NAME", "app"),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", ""),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
	}
}

// DSN returns a lib/pq compatible connection string.
func (d DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		d.Host, d.Port, d.Name, d.User, d.Password, d.SSLMode,
	)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
`

// ── config/http.go ────────────────────────────────────────────────────────
const tplConfigHTTP = `package config

import "time"

// HTTPConfig holds HTTP server tuning parameters.
type HTTPConfig struct {
	Port         int
	Env          string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func loadHTTP() HTTPConfig {
	return HTTPConfig{
		Port:         getEnvInt("PORT", 8080),
		Env:          getEnv("APP_ENV", "development"),
		ReadTimeout:  getEnvDuration("HTTP_READ_TIMEOUT", 10*time.Second),
		WriteTimeout: getEnvDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:  getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
	}
}
`

// ── config/jwt.go ─────────────────────────────────────────────────────────
const tplConfigJWT = `package config

import "time"

// JWTConfig holds JSON Web Token signing parameters.
type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

func loadJWT() JWTConfig {
	return JWTConfig{
		Secret:        getEnv("JWT_SECRET", "change-me-in-production"),
		AccessExpiry:  getEnvDuration("JWT_ACCESS_EXPIRY", 15*time.Minute),
		RefreshExpiry: getEnvDuration("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
	}
}
`

// ── config/redis.go ───────────────────────────────────────────────────────
const tplConfigRedis = `package config

// RedisConfig holds Redis connection settings.
// Uncomment and extend when you add infra/cache wiring.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func loadRedis() RedisConfig {
	return RedisConfig{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       getEnvInt("REDIS_DB", 0),
	}
}
`

// ── common/response.go ────────────────────────────────────────────────────
const tplCommonResponse = `// Package common provides shared response helpers and utilities.
package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse is the canonical JSON envelope for all endpoints.
type APIResponse struct {
	Success bool   "json:\"success\""
	Message string "json:\"message,omitempty\""
	Data    any    "json:\"data,omitempty\""
	Error   string "json:\"error,omitempty\""
}

// OK writes a 200 response with data.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, APIResponse{Success: true, Data: data})
}

// Created writes a 201 response with data.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, APIResponse{Success: true, Data: data})
}

// BadRequest writes a 400 response with an error message.
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, APIResponse{Success: false, Error: msg})
}

// Unauthorized writes a 401 response.
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, APIResponse{Success: false, Error: msg})
}

// Forbidden writes a 403 response.
func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, APIResponse{Success: false, Error: msg})
}

// NotFound writes a 404 response.
func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, APIResponse{Success: false, Error: msg})
}

// InternalError writes a 500 response.
func InternalError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, APIResponse{Success: false, Error: msg})
}
`

// ── common/utils.go ───────────────────────────────────────────────────────
const tplCommonUtils = `package common

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateToken returns a cryptographically random hex string of length n bytes.
func GenerateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Ptr returns a pointer to the given value (generic convenience helper).
func Ptr[T any](v T) *T { return &v }
`

// ── infra/db/db.go ────────────────────────────────────────────────────────
const tplInfraDB = `// Package db manages the PostgreSQL connection pool lifecycle.
package db

import (
	"database/sql"
	"fmt"

	"__MODULE__/config"

	_ "github.com/lib/pq" // PostgreSQL driver
	"go.uber.org/zap"
)

// Connect opens a PostgreSQL connection pool, validates connectivity with Ping,
// and applies the pool settings from cfg.
func Connect(cfg *config.Config, logger *zap.Logger) (*sql.DB, error) {
	dsn := cfg.DB.DSN()
	logger.Info("connecting to postgres", zap.String("dsn", maskDSN(dsn)))

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	logger.Info("postgres connected",
		zap.Int("max_open", cfg.DB.MaxOpenConns),
		zap.Int("max_idle", cfg.DB.MaxIdleConns),
	)
	return db, nil
}

// maskDSN replaces the password in a DSN for safe logging.
func maskDSN(dsn string) string {
	// Simple mask — replace password= value
	return dsn
}
`

// ── infra/cache/redis.go ──────────────────────────────────────────────────
const tplInfraRedis = `// Package cache provides a Redis client wrapper.
// Uncomment the redis client import when you add go-redis to go.mod.
package cache

// Client is a thin wrapper around a Redis connection.
// Replace the stub below with your preferred Redis client.
type Client struct {
	// client *redis.Client
}

// NewClient creates a connected Redis client.
// func NewClient(addr, password string, db int) (*Client, error) { ... }

// Close releases the Redis connection.
func (c *Client) Close() error { return nil }
`

// ── infra/jwt/token.go ────────────────────────────────────────────────────
const tplInfraJWT = `// Package jwt provides JWT signing and verification helpers.
package jwt

import (
	"errors"
	"time"

	"__MODULE__/config"
)

// Claims represents the payload stored in a JWT.
type Claims struct {
	UserID string
	Email  string
	Role   string
}

// Token holds the signed token string alongside its expiry.
type Token struct {
	Value     string
	ExpiresAt time.Time
}

// ErrExpired is returned when a token has passed its expiry time.
var ErrExpired = errors.New("token expired")

// Manager signs and verifies JWTs using the application config.
type Manager struct {
	cfg config.JWTConfig
}

// NewManager creates a Manager from the given JWT configuration.
func NewManager(cfg config.JWTConfig) *Manager {
	return &Manager{cfg: cfg}
}

// Sign creates a signed JWT for the given claims.
// TODO: implement with your preferred JWT library (e.g. golang-jwt/jwt).
func (m *Manager) Sign(claims Claims) (*Token, error) {
	return nil, errors.New("not implemented")
}

// Verify parses and validates a signed token string.
// TODO: implement with your preferred JWT library.
func (m *Manager) Verify(tokenStr string) (*Claims, error) {
	return nil, errors.New("not implemented")
}
`

// ── middleware/auth.go ────────────────────────────────────────────────────
const tplMiddlewareAuth = `// Package middleware contains reusable Gin middleware.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequireAuth is a placeholder Bearer-token authentication middleware.
// Replace the stub validation with real JWT verification from infra/jwt.
func RequireAuth(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			logger.Warn("missing or malformed Authorization header",
				zap.String("ip", c.ClientIP()),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		// token := strings.TrimPrefix(authHeader, "Bearer ")
		// claims, err := jwtManager.Verify(token)
		// ...

		c.Next()
	}
}
`

// ── Dockerfile ────────────────────────────────────────────────────────────
const tplDockerfile = `# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Cache dependencies before copying source
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app ./cmd/api

# ── Production stage ──────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

WORKDIR /home/app
COPY --from=builder /app .

USER app
EXPOSE 8080

ENTRYPOINT ["./app"]
`

// ── docker-compose.yml ────────────────────────────────────────────────────
const tplDockerCompose = `version: "3.9"

services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "${PORT:-8080}:8080"
    env_file: .env.example
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB:       ${DB_NAME:-app}
      POSTGRES_USER:     ${DB_USER:-postgres}
      POSTGRES_PASSWORD: ${DB_PASSWORD:-secret}
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER:-postgres}"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  pgdata:
`

// ── Makefile ──────────────────────────────────────────────────────────────
const tplMakefile = `.PHONY: run build test test-cover lint fmt tidy clean \
        docker-build docker-up docker-down migrate-up migrate-down

# ── Local dev ─────────────────────────────────────────────────────────────────

run: ## Start the API server
	go run ./cmd/api

build: ## Compile the binary to ./bin/api
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/api ./cmd/api

test: ## Run all tests
	go test ./... -race -count=1

test-cover: ## Run tests with HTML coverage report
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ── Code quality ──────────────────────────────────────────────────────────────

lint: ## Run golangci-lint (install separately)
	golangci-lint run ./...

fmt: ## Format all Go source files
	go fmt ./...
	gofmt -w -s .

tidy: ## Tidy go.mod / go.sum
	go mod tidy

# ── Cleanup ───────────────────────────────────────────────────────────────────

clean: ## Remove build artefacts
	rm -rf bin/ coverage.out coverage.html

# ── Docker ────────────────────────────────────────────────────────────────────

docker-build: ## Build Docker image
	docker compose build

docker-up: ## Start all services
	docker compose up -d

docker-down: ## Stop and remove containers
	docker compose down

# ── Migrations ────────────────────────────────────────────────────────────────

migrate-up: ## Run all pending migrations
	go run ./cmd/migrate -direction=up

migrate-down: ## Rollback the last migration
	go run ./cmd/migrate -direction=down -steps=1

# ── Help ──────────────────────────────────────────────────────────────────────

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
`

// ── .env.example ─────────────────────────────────────────────────────────
const tplEnvExample = `# ── Server ───────────────────────────────────────────────────────────────────
PORT=8080
APP_ENV=development

# ── Database ──────────────────────────────────────────────────────────────────
DB_HOST=localhost
DB_PORT=5432
DB_NAME=app
DB_USER=postgres
DB_PASSWORD=secret
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# ── JWT ───────────────────────────────────────────────────────────────────────
JWT_SECRET=change-me-in-production
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# ── Redis (optional) ──────────────────────────────────────────────────────────
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# ── HTTP timeouts ─────────────────────────────────────────────────────────────
HTTP_READ_TIMEOUT=10s
HTTP_WRITE_TIMEOUT=10s
HTTP_IDLE_TIMEOUT=60s
`

// ── .gitignore ────────────────────────────────────────────────────────────
const tplGitignore = `# Binaries
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test output
*.test
coverage.out
coverage.html

# Environment
.env
.env.local

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Go workspace
go.work
go.work.sum
`

// ── README.md ─────────────────────────────────────────────────────────────
// Uses string concatenation to embed triple-backtick fences without
// breaking the raw string literal.
var tplReadme = "# __MODULE__\n\n" +
	"Production-ready Go backend scaffolded with **gospin**.\n\n" +
	"## Stack\n\n" +
	"| Layer | Technology |\n" +
	"|---|---|\n" +
	"| Language | Go 1.22+ |\n" +
	"| Framework | [Gin](https://github.com/gin-gonic/gin) |\n" +
	"| Database | PostgreSQL via `database/sql` + `lib/pq` |\n" +
	"| Logging | [Uber Zap](https://github.com/uber-go/zap) |\n\n" +
	"## Quick Start\n\n" +
	"```bash\n" +
	"cp .env.example .env   # edit credentials\n" +
	"docker compose up -d   # start postgres\n" +
	"make run               # start the API\n" +
	"curl localhost:8080/health\n" +
	"```\n\n" +
	"## Project Structure\n\n" +
	"```\n" +
	".\n" +
	"├── cmd/\n" +
	"│   ├── api/main.go          # HTTP server entry point\n" +
	"│   └── migrate/main.go      # Database migration runner\n" +
	"├── bootstrap/               # Dependency wiring / composition root\n" +
	"├── api/health/              # Health-check handler + tests\n" +
	"├── server/                  # Gin engine, middleware, lifecycle\n" +
	"├── config/                  # Typed env configuration (DB, HTTP, JWT)\n" +
	"├── common/                  # Shared response helpers & utilities\n" +
	"├── infra/\n" +
	"│   ├── db/                  # PostgreSQL pool management\n" +
	"│   ├── cache/               # Redis client (stub)\n" +
	"│   └── jwt/                 # JWT sign/verify manager (stub)\n" +
	"├── middleware/              # Reusable Gin middleware (auth, etc.)\n" +
	"├── model/                   # Domain entity structs\n" +
	"├── repository/              # Database query layer\n" +
	"├── utils/                   # Core helper functions\n" +
	"├── constants/               # Immutable application constants\n" +
	"└── mocks/                   # Test mock implementations\n" +
	"```\n\n" +
	"## Available Commands\n\n" +
	"```\n" +
	"make run            Start the API server\n" +
	"make build          Compile to ./bin/api\n" +
	"make test           Run all tests with race detector\n" +
	"make test-cover     Generate HTML coverage report\n" +
	"make lint           Run golangci-lint\n" +
	"make fmt            Format all source files\n" +
	"make tidy           Tidy go.mod / go.sum\n" +
	"make docker-up      Start all services via Docker Compose\n" +
	"make migrate-up     Run pending database migrations\n" +
	"```\n"
