package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	_ "github.com/lib/pq"
	"myconnectionsvr/modern-mcs/internal/audit"
	"myconnectionsvr/modern-mcs/internal/auth"
	"myconnectionsvr/modern-mcs/internal/config"
	"myconnectionsvr/modern-mcs/internal/httpserver"
	"myconnectionsvr/modern-mcs/internal/migrations"
	"myconnectionsvr/modern-mcs/internal/observability"
	"myconnectionsvr/modern-mcs/internal/sqlprofile"
)

type App struct {
	cfg    config.Config
	log    *slog.Logger
	db     *sql.DB
	server *httpserver.Server
}

func New(cfg config.Config) (*App, error) {
	logger := observability.NewLogger()

	var err error
	var db *sql.DB
	if cfg.DatabaseURL != "" {
		db, err = sql.Open("postgres", cfg.DatabaseURL)
		if err != nil {
			return nil, fmt.Errorf("open database: %w", err)
		}
		if err := db.Ping(); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("ping database: %w", err)
		}
	}

	var userStore auth.UserStore
	var sessionStore auth.SessionStore
	if db != nil {
		userStore, err = auth.NewPostgresUserStore(db)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("create postgres user store: %w", err)
		}
		sessionStore, err = auth.NewPostgresSessionStore(db)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("create postgres session store: %w", err)
		}
	} else {
		userStore, err = auth.NewFileUserStore(cfg.Auth.UserStateFile)
		if err != nil {
			return nil, fmt.Errorf("create user store: %w", err)
		}
	}
	authService, err := auth.NewService(userStore, auth.ServiceConfig{
		PasswordPepper:   cfg.Auth.PasswordPepper,
		SessionTTL:       cfg.Auth.SessionTTL,
		SessionStateFile: cfg.Auth.SessionStateFile,
		SessionStore:     sessionStore,
	})
	if err != nil {
		return nil, fmt.Errorf("create auth service: %w", err)
	}
	if err := authService.LoadSessionState(); err != nil {
		if db != nil {
			_ = db.Close()
		}
		return nil, fmt.Errorf("load auth session state: %w", err)
	}

	if _, err := userStore.GetByUsername(cfg.Auth.BootstrapUsername); err != nil {
		if errors.Is(err, auth.ErrUserNotFound) {
			if err := userStore.Put(auth.User{
				ID:           "bootstrap-admin",
				Username:     cfg.Auth.BootstrapUsername,
				PasswordHash: authService.HashPassword(cfg.Auth.BootstrapPassword),
				Roles:        []string{"admin"},
			}); err != nil {
				if db != nil {
					_ = db.Close()
				}
				return nil, fmt.Errorf("create bootstrap user: %w", err)
			}
			logger.Info("bootstrap auth user created", "username", cfg.Auth.BootstrapUsername)
		} else {
			if db != nil {
				_ = db.Close()
			}
			return nil, fmt.Errorf("check bootstrap user: %w", err)
		}
	}

	var sqlProfileService httpserver.SQLProfileService
	if db != nil {
		sqlProfileService, err = sqlprofile.NewPGService(db)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("create postgres sql profile service: %w", err)
		}
	} else {
		sqlProfileService, err = sqlprofile.NewServiceWithFile(cfg.SQLProfileStateFile)
		if err != nil {
			return nil, fmt.Errorf("create sql profile service: %w", err)
		}
	}

	var migrationService httpserver.MigrationService
	if db != nil {
		migrationService, err = migrations.NewServiceWithPostgres(cfg.MigrationsDir, db)
		if err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("create postgres migration service: %w", err)
		}
	} else {
		migrationService = migrations.NewService(cfg.MigrationsDir, cfg.MigrationStateFile)
	}
	auditLogger := audit.NewLogger(cfg.AuditLogFile)

	server := httpserver.New(cfg.HTTP, httpserver.Deps{
		Auth:            authService,
		SQLProfiles:     sqlProfileService,
		Migrations:      migrationService,
		Audit:           auditLogger,
		FrontendDistDir: cfg.FrontendDistDir,
	})

	return &App{
		cfg:    cfg,
		log:    logger,
		db:     db,
		server: server,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer func() {
		if a.db != nil {
			_ = a.db.Close()
		}
	}()

	errCh := make(chan error, 1)

	go func() {
		a.log.Info("http server starting", "addr", a.cfg.HTTP.Addr)
		errCh <- a.server.Start()
	}()

	select {
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.HTTP.ShutdownTimeout)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("server exited: %w", err)
	}
}
