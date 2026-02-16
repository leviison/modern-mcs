package httpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"myconnectionsvr/modern-mcs/internal/auth"
	"myconnectionsvr/modern-mcs/internal/migrations"
	"myconnectionsvr/modern-mcs/internal/sqlprofile"
)

type fakeAuthService struct {
	loginFunc             func(username, password string) (auth.Session, error)
	validateFunc          func(token string) (auth.Session, error)
	logoutFunc            func(token string) error
	changePasswordFunc    func(token, currentPassword, newPassword string) error
	listSessionViewsFunc  func() []auth.SessionView
	revokeSessionByIDFunc func(sessionID string) error
}

func (f fakeAuthService) Login(username, password string) (auth.Session, error) {
	if f.loginFunc == nil {
		return auth.Session{}, errors.New("not implemented")
	}
	return f.loginFunc(username, password)
}

func (f fakeAuthService) ValidateToken(token string) (auth.Session, error) {
	if f.validateFunc == nil {
		return auth.Session{}, errors.New("not implemented")
	}
	return f.validateFunc(token)
}

func (f fakeAuthService) Logout(token string) error {
	if f.logoutFunc == nil {
		return errors.New("not implemented")
	}
	return f.logoutFunc(token)
}

func (f fakeAuthService) ChangePassword(token, currentPassword, newPassword string) error {
	if f.changePasswordFunc == nil {
		return errors.New("not implemented")
	}
	return f.changePasswordFunc(token, currentPassword, newPassword)
}

func (f fakeAuthService) ListSessionViews() []auth.SessionView {
	if f.listSessionViewsFunc == nil {
		return nil
	}
	return f.listSessionViewsFunc()
}

func (f fakeAuthService) RevokeSessionByID(sessionID string) error {
	if f.revokeSessionByIDFunc == nil {
		return errors.New("not implemented")
	}
	return f.revokeSessionByIDFunc(sessionID)
}

type fakeSQLProfileService struct {
	listFunc   func() []sqlprofile.Profile
	createFunc func(p sqlprofile.Profile) (sqlprofile.Profile, error)
	getFunc    func(id string) (sqlprofile.Profile, error)
	updateFunc func(id string, p sqlprofile.Profile) (sqlprofile.Profile, error)
	deleteFunc func(id string) error
}

func (f fakeSQLProfileService) Create(p sqlprofile.Profile) (sqlprofile.Profile, error) {
	return f.createFunc(p)
}
func (f fakeSQLProfileService) List() []sqlprofile.Profile { return f.listFunc() }
func (f fakeSQLProfileService) Get(id string) (sqlprofile.Profile, error) {
	return f.getFunc(id)
}
func (f fakeSQLProfileService) Update(id string, p sqlprofile.Profile) (sqlprofile.Profile, error) {
	return f.updateFunc(id, p)
}
func (f fakeSQLProfileService) Delete(id string) error { return f.deleteFunc(id) }

type fakeMigrationService struct {
	listFunc        func() ([]migrations.FileInfo, error)
	statusFunc      func() ([]migrations.Status, error)
	markAppliedFunc func(name string, appliedAt time.Time) error
}

func (f fakeMigrationService) List() ([]migrations.FileInfo, error) { return f.listFunc() }
func (f fakeMigrationService) Status() ([]migrations.Status, error) { return f.statusFunc() }
func (f fakeMigrationService) MarkApplied(name string, appliedAt time.Time) error {
	return f.markAppliedFunc(name, appliedAt)
}

func TestHealthz(t *testing.T) {
	handler := loggingMiddleware(NewHandler(Deps{}))
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatalf("expected X-Request-Id header to be set")
	}
}

func TestInfo(t *testing.T) {
	handler := NewHandler(Deps{})
	req := httptest.NewRequest(http.MethodGet, "/v1/info", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if got["service"] != "modern-mcs-api" {
		t.Fatalf("expected service 'modern-mcs-api', got %q", got["service"])
	}
}

func TestLoginSuccess(t *testing.T) {
	handler := NewHandler(Deps{Auth: fakeAuthService{loginFunc: func(username, password string) (auth.Session, error) {
		if username != "admin" || password != "secret" {
			return auth.Session{}, auth.ErrInvalidCredentials
		}
		return auth.Session{ID: "s1", Token: "token-123", UserID: "u-1", Username: "admin", Roles: []string{"admin"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
	}, validateFunc: func(token string) (auth.Session, error) {
		return auth.Session{}, errors.New("not used")
	}, logoutFunc: func(token string) error { return errors.New("not used") }}})

	body := bytes.NewBufferString(`{"username":"admin","password":"secret"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if got["session_id"] != "s1" {
		t.Fatalf("expected session_id s1, got %v", got["session_id"])
	}
}

func TestAuthMeSuccess(t *testing.T) {
	handler := NewHandler(Deps{Auth: fakeAuthService{loginFunc: func(_, _ string) (auth.Session, error) {
		return auth.Session{}, errors.New("not used")
	}, validateFunc: func(token string) (auth.Session, error) {
		if token != "token-123" {
			return auth.Session{}, auth.ErrInvalidToken
		}
		return auth.Session{UserID: "u-1", Username: "admin", Roles: []string{"admin"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
	}, logoutFunc: func(token string) error { return errors.New("not used") }}})

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestAuthLogout(t *testing.T) {
	handler := NewHandler(Deps{Auth: fakeAuthService{
		loginFunc: func(_, _ string) (auth.Session, error) { return auth.Session{}, errors.New("not used") },
		validateFunc: func(token string) (auth.Session, error) {
			return auth.Session{}, errors.New("not used")
		},
		logoutFunc: func(token string) error {
			if token != "token-123" {
				return auth.ErrInvalidToken
			}
			return nil
		},
	}})

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}

	reqBad := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	reqBad.Header.Set("Authorization", "Bearer bad-token")
	recBad := httptest.NewRecorder()
	handler.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", recBad.Code)
	}
}

func TestAuthChangePassword(t *testing.T) {
	handler := NewHandler(Deps{Auth: fakeAuthService{
		changePasswordFunc: func(token, currentPassword, newPassword string) error {
			if token != "token-123" {
				return auth.ErrInvalidToken
			}
			if currentPassword != "oldpass123" || newPassword != "NewPassword123!" {
				return auth.ErrInvalidCredentials
			}
			return nil
		},
	}})

	body := bytes.NewBufferString(`{"current_password":"oldpass123","new_password":"NewPassword123!"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/change-password", body)
	req.Header.Set("Authorization", "Bearer token-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d body=%s", rec.Code, rec.Body.String())
	}

	bad := bytes.NewBufferString(`{"current_password":"oldpass123","new_password":"short"}`)
	reqBad := httptest.NewRequest(http.MethodPost, "/v1/auth/change-password", bad)
	reqBad.Header.Set("Authorization", "Bearer token-123")
	handlerBad := NewHandler(Deps{Auth: fakeAuthService{
		changePasswordFunc: func(token, currentPassword, newPassword string) error { return auth.ErrWeakPassword },
	}})
	recBad := httptest.NewRecorder()
	handlerBad.ServeHTTP(recBad, reqBad)
	if recBad.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recBad.Code)
	}
}

func TestSQLProfilesUnauthorized(t *testing.T) {
	handler := NewHandler(Deps{})
	req := httptest.NewRequest(http.MethodGet, "/v1/sql-profiles", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 for missing auth service, got %d", rec.Code)
	}
}

func TestSQLProfilesListAuthorized(t *testing.T) {
	handler := NewHandler(Deps{
		Auth: fakeAuthService{loginFunc: nil, validateFunc: func(token string) (auth.Session, error) {
			if token != "admin-token" {
				return auth.Session{}, auth.ErrInvalidToken
			}
			return auth.Session{UserID: "u-1", Username: "admin", Roles: []string{"admin"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
		}, logoutFunc: func(token string) error { return errors.New("not used") }},
		SQLProfiles: fakeSQLProfileService{
			listFunc: func() []sqlprofile.Profile {
				return []sqlprofile.Profile{{ID: "p1", Name: "Main", DBType: "mysql", Host: "db", Port: 3306, Database: "mcs", Commands: "SELECT 1"}}
			},
			createFunc: func(p sqlprofile.Profile) (sqlprofile.Profile, error) {
				return sqlprofile.Profile{}, errors.New("not used")
			},
			getFunc: func(id string) (sqlprofile.Profile, error) { return sqlprofile.Profile{}, errors.New("not used") },
			updateFunc: func(id string, p sqlprofile.Profile) (sqlprofile.Profile, error) {
				return sqlprofile.Profile{}, errors.New("not used")
			},
			deleteFunc: func(id string) error { return errors.New("not used") },
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/sql-profiles", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	items, ok := got["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one sql profile item")
	}
}

func TestMigrationsListAuthorized(t *testing.T) {
	handler := NewHandler(Deps{
		Auth: fakeAuthService{validateFunc: func(token string) (auth.Session, error) {
			return auth.Session{UserID: "u-1", Username: "admin", Roles: []string{"admin"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
		}, loginFunc: func(username, password string) (auth.Session, error) { return auth.Session{}, errors.New("not used") }, logoutFunc: func(token string) error { return errors.New("not used") }},
		Migrations: fakeMigrationService{listFunc: func() ([]migrations.FileInfo, error) {
			return []migrations.FileInfo{{Name: "0001_init.sql", Checksum: "abc"}}, nil
		}, statusFunc: func() ([]migrations.Status, error) {
			return nil, errors.New("not used")
		}, markAppliedFunc: func(name string, appliedAt time.Time) error {
			return errors.New("not used")
		}},
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/system/migrations", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminSessionListAndRevoke(t *testing.T) {
	revokeCalled := false
	handler := NewHandler(Deps{
		Auth: fakeAuthService{
			validateFunc: func(token string) (auth.Session, error) {
				return auth.Session{UserID: "u-1", Username: "admin", Roles: []string{"admin"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
			listSessionViewsFunc: func() []auth.SessionView {
				return []auth.SessionView{{ID: "s1", UserID: "u-1", Username: "admin", Roles: []string{"admin"}, ExpiresAt: time.Now().Add(time.Hour)}}
			},
			revokeSessionByIDFunc: func(sessionID string) error {
				if sessionID != "s1" {
					return auth.ErrInvalidToken
				}
				revokeCalled = true
				return nil
			},
		},
	})

	reqList := httptest.NewRequest(http.MethodGet, "/v1/system/sessions", nil)
	reqList.Header.Set("Authorization", "Bearer admin-token")
	recList := httptest.NewRecorder()
	handler.ServeHTTP(recList, reqList)
	if recList.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recList.Code, recList.Body.String())
	}

	reqDel := httptest.NewRequest(http.MethodDelete, "/v1/system/sessions/s1", nil)
	reqDel.Header.Set("Authorization", "Bearer admin-token")
	recDel := httptest.NewRecorder()
	handler.ServeHTTP(recDel, reqDel)
	if recDel.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d body=%s", recDel.Code, recDel.Body.String())
	}
	if !revokeCalled {
		t.Fatalf("expected revoke session by id to be called")
	}
}

func TestMigrationsStatusAndApplyAuthorized(t *testing.T) {
	applyCalled := false
	handler := NewHandler(Deps{
		Auth: fakeAuthService{
			validateFunc: func(token string) (auth.Session, error) {
				return auth.Session{UserID: "u-1", Username: "admin", Roles: []string{"admin"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
			},
			loginFunc: func(username, password string) (auth.Session, error) {
				return auth.Session{}, errors.New("not used")
			},
			logoutFunc: func(token string) error { return errors.New("not used") },
		},
		Migrations: fakeMigrationService{
			listFunc: func() ([]migrations.FileInfo, error) { return nil, errors.New("not used") },
			statusFunc: func() ([]migrations.Status, error) {
				return []migrations.Status{{Name: "0001_init.sql", Applied: true, AppliedAt: "2026-02-16T12:00:00Z"}}, nil
			},
			markAppliedFunc: func(name string, appliedAt time.Time) error {
				if name != "0002_more.sql" {
					t.Fatalf("expected name 0002_more.sql, got %q", name)
				}
				applyCalled = true
				return nil
			},
		},
	})

	reqStatus := httptest.NewRequest(http.MethodGet, "/v1/system/migrations/status", nil)
	reqStatus.Header.Set("Authorization", "Bearer admin-token")
	recStatus := httptest.NewRecorder()
	handler.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recStatus.Code)
	}

	reqApply := httptest.NewRequest(http.MethodPost, "/v1/system/migrations/0002_more.sql/apply", nil)
	reqApply.Header.Set("Authorization", "Bearer admin-token")
	recApply := httptest.NewRecorder()
	handler.ServeHTTP(recApply, reqApply)
	if recApply.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recApply.Code, recApply.Body.String())
	}
	if !applyCalled {
		t.Fatalf("expected mark applied to be called")
	}
}

func TestUnknownRoute(t *testing.T) {
	handler := NewHandler(Deps{})
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
}

func TestFrontendStaticAndSpaFallback(t *testing.T) {
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("<html>app</html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dist, "app.js"), []byte("console.log('x')"), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	handler := NewHandler(Deps{FrontendDistDir: dist})

	recRoot := httptest.NewRecorder()
	handler.ServeHTTP(recRoot, httptest.NewRequest(http.MethodGet, "/", nil))
	if recRoot.Code != http.StatusOK {
		t.Fatalf("expected root 200, got %d", recRoot.Code)
	}

	recAsset := httptest.NewRecorder()
	handler.ServeHTTP(recAsset, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if recAsset.Code != http.StatusOK {
		t.Fatalf("expected asset 200, got %d", recAsset.Code)
	}

	recSpa := httptest.NewRecorder()
	handler.ServeHTTP(recSpa, httptest.NewRequest(http.MethodGet, "/some/spa/route", nil))
	if recSpa.Code != http.StatusOK {
		t.Fatalf("expected spa fallback 200, got %d", recSpa.Code)
	}

	recAPI := httptest.NewRecorder()
	handler.ServeHTTP(recAPI, httptest.NewRequest(http.MethodGet, "/v1/not-found", nil))
	if recAPI.Code != http.StatusNotFound {
		t.Fatalf("expected API not shadowed, got status %d", recAPI.Code)
	}
}
