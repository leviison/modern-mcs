package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"myconnectionsvr/modern-mcs/internal/auth"
	"myconnectionsvr/modern-mcs/internal/config"
	"myconnectionsvr/modern-mcs/internal/migrations"
	"myconnectionsvr/modern-mcs/internal/sqlprofile"
)

type AuthService interface {
	Login(username, password string) (auth.Session, error)
	ValidateToken(token string) (auth.Session, error)
	Logout(token string) error
	ChangePassword(token, currentPassword, newPassword string) error
	ListSessionViews() []auth.SessionView
	RevokeSessionByID(sessionID string) error
}

type SQLProfileService interface {
	Create(p sqlprofile.Profile) (sqlprofile.Profile, error)
	List() []sqlprofile.Profile
	Get(id string) (sqlprofile.Profile, error)
	Update(id string, p sqlprofile.Profile) (sqlprofile.Profile, error)
	Delete(id string) error
}

type MigrationService interface {
	List() ([]migrations.FileInfo, error)
	Status() ([]migrations.Status, error)
	MarkApplied(name string, appliedAt time.Time) error
}

type AuditLogger interface {
	Log(actor, action, target, outcome, detail string) error
}

type Deps struct {
	Auth            AuthService
	SQLProfiles     SQLProfileService
	Migrations      MigrationService
	Audit           AuditLogger
	FrontendDistDir string
}

type Server struct {
	httpServer *http.Server
}

func New(cfg config.HTTPConfig, deps Deps) *Server {
	handler := NewHandler(deps)

	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Addr,
			Handler:      loggingMiddleware(handler),
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  60 * time.Second,
		},
	}
}

func NewHandler(deps Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("/v1/info", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"service": "modern-mcs-api",
			"version": "0.1.0",
		})
	})

	registerAuthHandlers(mux, deps)
	registerSessionAdminHandlers(mux, deps)
	registerSQLProfileHandlers(mux, deps)
	registerMigrationHandlers(mux, deps)
	registerFrontendHandlers(mux, deps.FrontendDistDir)

	return mux
}

func registerAuthHandlers(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if deps.Auth == nil {
			writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
			return
		}

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Username == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "username and password are required")
			return
		}

		session, err := deps.Auth.Login(req.Username, req.Password)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				auditReq(deps.Audit, r, req.Username, "auth.login", "", "failed", "", "invalid credentials")
				writeError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}
			auditReq(deps.Audit, r, req.Username, "auth.login", "", "failed", "", err.Error())
			writeError(w, http.StatusInternalServerError, "login failed")
			return
		}
		auditReq(deps.Audit, r, session.Username, "auth.login", "", "success", session.ID, "")

		writeJSON(w, http.StatusOK, map[string]any{
			"token":      session.Token,
			"session_id": session.ID,
			"user": map[string]any{
				"id":       session.UserID,
				"username": session.Username,
				"roles":    session.Roles,
			},
			"expires_at": session.ExpiresAt.UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/v1/auth/me", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		session, ok := requireSession(w, r, deps.Auth, "")
		if !ok {
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"id":         session.UserID,
			"username":   session.Username,
			"roles":      session.Roles,
			"expires_at": session.ExpiresAt.UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if deps.Auth == nil {
			writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
			return
		}
		token, err := extractBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
		}
		session, _ := deps.Auth.ValidateToken(token)
		if err := deps.Auth.Logout(token); err != nil {
			auditReq(deps.Audit, r, session.Username, "auth.logout", "", "failed", session.ID, "invalid token")
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		auditReq(deps.Audit, r, session.Username, "auth.logout", "", "success", session.ID, "")
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/v1/auth/change-password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if deps.Auth == nil {
			writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
			return
		}
		token, err := extractBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
		}
		session, _ := deps.Auth.ValidateToken(token)

		var req struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.CurrentPassword == "" || req.NewPassword == "" {
			writeError(w, http.StatusBadRequest, "current_password and new_password are required")
			return
		}

		if err := deps.Auth.ChangePassword(token, req.CurrentPassword, req.NewPassword); err != nil {
			if errors.Is(err, auth.ErrWeakPassword) {
				auditReq(deps.Audit, r, session.Username, "auth.change_password", "", "failed", session.ID, "weak password")
				writeError(w, http.StatusBadRequest, "new password does not meet policy")
				return
			}
			if errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, auth.ErrInvalidCredentials) {
				auditReq(deps.Audit, r, session.Username, "auth.change_password", "", "failed", session.ID, "invalid credentials or token")
				writeError(w, http.StatusUnauthorized, "invalid credentials or token")
				return
			}
			auditReq(deps.Audit, r, session.Username, "auth.change_password", "", "failed", session.ID, err.Error())
			writeError(w, http.StatusInternalServerError, "change password failed")
			return
		}
		auditReq(deps.Audit, r, session.Username, "auth.change_password", "", "success", session.ID, "")
		w.WriteHeader(http.StatusNoContent)
	})
}

func registerSessionAdminHandlers(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("/v1/system/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		adminSession, ok := requireSession(w, r, deps.Auth, "admin")
		if !ok {
			return
		}
		if deps.Auth == nil {
			writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": deps.Auth.ListSessionViews()})
		auditReq(deps.Audit, r, adminSession.Username, "session.list", "", "success", adminSession.ID, "")
	})

	mux.HandleFunc("/v1/system/sessions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		adminSession, ok := requireSession(w, r, deps.Auth, "admin")
		if !ok {
			return
		}
		if deps.Auth == nil {
			writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
			return
		}

		sessionID := strings.TrimPrefix(r.URL.Path, "/v1/system/sessions/")
		sessionID = strings.TrimSpace(sessionID)
		if sessionID == "" || strings.Contains(sessionID, "/") {
			writeError(w, http.StatusBadRequest, "invalid session id")
			return
		}
		err := deps.Auth.RevokeSessionByID(sessionID)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidToken) {
				auditReq(deps.Audit, r, adminSession.Username, "session.revoke", sessionID, "failed", adminSession.ID, "session not found")
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			auditReq(deps.Audit, r, adminSession.Username, "session.revoke", sessionID, "failed", adminSession.ID, err.Error())
			writeError(w, http.StatusInternalServerError, "revoke session failed")
			return
		}
		auditReq(deps.Audit, r, adminSession.Username, "session.revoke", sessionID, "success", adminSession.ID, "")
		w.WriteHeader(http.StatusNoContent)
	})
}

func registerSQLProfileHandlers(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("/v1/sql-profiles", func(w http.ResponseWriter, r *http.Request) {
		adminSession, ok := requireSession(w, r, deps.Auth, "admin")
		if !ok {
			return
		}
		if deps.SQLProfiles == nil {
			writeError(w, http.StatusServiceUnavailable, "sql profile service unavailable")
			return
		}

		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, map[string]any{"items": deps.SQLProfiles.List()})
		case http.MethodPost:
			var req sqlprofile.Profile
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			created, err := deps.SQLProfiles.Create(req)
			if err != nil {
				if errors.Is(err, sqlprofile.ErrInvalidInput) {
					writeError(w, http.StatusBadRequest, err.Error())
					return
				}
				writeError(w, http.StatusInternalServerError, "create profile failed")
				return
			}
			auditReq(deps.Audit, r, adminSession.Username, "sqlprofile.create", created.ID, "success", adminSession.ID, "")
			writeJSON(w, http.StatusCreated, created)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	mux.HandleFunc("/v1/sql-profiles/", func(w http.ResponseWriter, r *http.Request) {
		adminSession, ok := requireSession(w, r, deps.Auth, "admin")
		if !ok {
			return
		}
		if deps.SQLProfiles == nil {
			writeError(w, http.StatusServiceUnavailable, "sql profile service unavailable")
			return
		}

		id := strings.TrimPrefix(r.URL.Path, "/v1/sql-profiles/")
		if id == "" || strings.Contains(id, "/") {
			writeError(w, http.StatusNotFound, "profile not found")
			return
		}

		switch r.Method {
		case http.MethodGet:
			p, err := deps.SQLProfiles.Get(id)
			if err != nil {
				if errors.Is(err, sqlprofile.ErrNotFound) {
					writeError(w, http.StatusNotFound, "profile not found")
					return
				}
				writeError(w, http.StatusInternalServerError, "get profile failed")
				return
			}
			writeJSON(w, http.StatusOK, p)
		case http.MethodPut:
			var req sqlprofile.Profile
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			updated, err := deps.SQLProfiles.Update(id, req)
			if err != nil {
				if errors.Is(err, sqlprofile.ErrInvalidInput) {
					writeError(w, http.StatusBadRequest, err.Error())
					return
				}
				if errors.Is(err, sqlprofile.ErrNotFound) {
					writeError(w, http.StatusNotFound, "profile not found")
					return
				}
				writeError(w, http.StatusInternalServerError, "update profile failed")
				return
			}
			auditReq(deps.Audit, r, adminSession.Username, "sqlprofile.update", updated.ID, "success", adminSession.ID, "")
			writeJSON(w, http.StatusOK, updated)
		case http.MethodDelete:
			err := deps.SQLProfiles.Delete(id)
			if err != nil {
				if errors.Is(err, sqlprofile.ErrNotFound) {
					writeError(w, http.StatusNotFound, "profile not found")
					return
				}
				writeError(w, http.StatusInternalServerError, "delete profile failed")
				return
			}
			auditReq(deps.Audit, r, adminSession.Username, "sqlprofile.delete", id, "success", adminSession.ID, "")
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
}

func registerMigrationHandlers(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("/v1/system/migrations", func(w http.ResponseWriter, r *http.Request) {
		adminSession, ok := requireSession(w, r, deps.Auth, "admin")
		if !ok {
			return
		}
		if deps.Migrations == nil {
			writeError(w, http.StatusServiceUnavailable, "migration service unavailable")
			return
		}

		switch r.Method {
		case http.MethodGet:
			files, err := deps.Migrations.List()
			if err != nil {
				writeError(w, http.StatusInternalServerError, "list migrations failed")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"items": files})
			auditReq(deps.Audit, r, adminSession.Username, "migration.list", "", "success", adminSession.ID, "")
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	mux.HandleFunc("/v1/system/migrations/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		adminSession, ok := requireSession(w, r, deps.Auth, "admin")
		if !ok {
			return
		}
		if deps.Migrations == nil {
			writeError(w, http.StatusServiceUnavailable, "migration service unavailable")
			return
		}
		status, err := deps.Migrations.Status()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "migration status failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": status})
		auditReq(deps.Audit, r, adminSession.Username, "migration.status", "", "success", adminSession.ID, "")
	})

	mux.HandleFunc("/v1/system/migrations/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		adminSession, ok := requireSession(w, r, deps.Auth, "admin")
		if !ok {
			return
		}
		if deps.Migrations == nil {
			writeError(w, http.StatusServiceUnavailable, "migration service unavailable")
			return
		}

		trimmed := strings.TrimPrefix(r.URL.Path, "/v1/system/migrations/")
		if !strings.HasSuffix(trimmed, "/apply") {
			writeError(w, http.StatusNotFound, "migration route not found")
			return
		}
		name := strings.TrimSuffix(trimmed, "/apply")
		name = strings.TrimSuffix(name, "/")
		if name == "" || strings.Contains(name, "/") {
			writeError(w, http.StatusBadRequest, "invalid migration name")
			return
		}

		if err := deps.Migrations.MarkApplied(name, time.Now()); err != nil {
			auditReq(deps.Audit, r, adminSession.Username, "migration.apply", name, "failed", adminSession.ID, err.Error())
			writeError(w, http.StatusBadRequest, "mark migration applied failed")
			return
		}
		auditReq(deps.Audit, r, adminSession.Username, "migration.apply", name, "success", adminSession.ID, "")
		writeJSON(w, http.StatusOK, map[string]string{"status": "applied", "name": name})
	})
}

func registerFrontendHandlers(mux *http.ServeMux, distDir string) {
	distDir = strings.TrimSpace(distDir)
	if distDir == "" {
		return
	}
	indexPath := filepath.Join(distDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return
	}

	fileServer := http.FileServer(http.Dir(distDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1/") || r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			http.NotFound(w, r)
			return
		}

		cleanPath := path.Clean(r.URL.Path)
		if cleanPath == "." || cleanPath == "/" {
			http.ServeFile(w, r, indexPath)
			return
		}

		fullPath := filepath.Join(distDir, strings.TrimPrefix(cleanPath, "/"))
		info, err := os.Stat(fullPath)
		if err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback.
		http.ServeFile(w, r, indexPath)
	})
}

func requireSession(w http.ResponseWriter, r *http.Request, authSvc AuthService, requiredRole string) (auth.Session, bool) {
	if authSvc == nil {
		writeError(w, http.StatusServiceUnavailable, "auth service unavailable")
		return auth.Session{}, false
	}
	token, err := extractBearerToken(r.Header.Get("Authorization"))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return auth.Session{}, false
	}

	session, err := authSvc.ValidateToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return auth.Session{}, false
	}

	if requiredRole != "" && !hasRole(session.Roles, requiredRole) {
		writeError(w, http.StatusForbidden, "forbidden")
		return auth.Session{}, false
	}

	return session, true
}

func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if strings.EqualFold(strings.TrimSpace(r), role) {
			return true
		}
	}
	return false
}

func extractBearerToken(authHeader string) (string, error) {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("invalid authorization header")
	}
	return strings.TrimSpace(parts[1]), nil
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := strings.TrimSpace(r.Header.Get("X-Request-Id"))
		if reqID == "" {
			reqID = newRequestID()
		}
		w.Header().Set("X-Request-Id", reqID)
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey{}, reqID))
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
	})
}

type requestIDKey struct{}

func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func requestIDFromContext(ctx context.Context) string {
	v := ctx.Value(requestIDKey{})
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func clientIP(r *http.Request) string {
	if fwd := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func auditReq(a AuditLogger, r *http.Request, actor, action, target, outcome, sessionID, detail string) {
	parts := []string{
		"rid=" + requestIDFromContext(r.Context()),
		"ip=" + clientIP(r),
		"ua=" + strings.TrimSpace(r.UserAgent()),
	}
	if sessionID != "" {
		parts = append(parts, "sid="+sessionID)
	}
	if strings.TrimSpace(detail) != "" {
		parts = append(parts, "detail="+strings.TrimSpace(detail))
	}
	auditSafe(a, actor, action, target, outcome, strings.Join(parts, " | "))
}

func auditSafe(a AuditLogger, actor, action, target, outcome, detail string) {
	if a == nil {
		return
	}
	_ = a.Log(actor, action, target, outcome, detail)
}
