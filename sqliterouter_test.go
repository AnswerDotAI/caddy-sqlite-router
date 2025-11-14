package sqliterouter

import (
	"context"
	"testing"
	"net/http"
	"net/http/httptest"	
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

func TestUnmarshalCaddyfile(t *testing.T) {
	dbPath := "test.db"
	query := "SELECT host, port FROM route WHERE subdomain = ?"
	config := "sqlite_router " + dbPath + " " + `"` + query + `"`
	dispenser := caddyfile.NewTestDispenser(config)
	sr := new(SQLiteRouter)
	err := sr.UnmarshalCaddyfile(dispenser)
	if err != nil { t.Errorf("UnmarshalCaddyfile failed with %v", err); return }
	if sr.DBPath != dbPath { t.Errorf("Expected DBPath to be '%s' but got '%s'", dbPath, sr.DBPath) }
	if sr.Query != query { t.Errorf("Expected Query to be '%s' but got '%s'", query, sr.Query) }
}

func TestProvision(t *testing.T) {
	sr := &SQLiteRouter{DBPath: "test.db", Query: "SELECT host, port FROM route WHERE domain = ?"}
	if err := sr.Provision(caddy.Context{}); err != nil { t.Errorf("Provision failed: %v", err) }
	if sr.db == nil { t.Error("Expected db to be initialized after Provision") }
}

func TestProvisionInvalidDB(t *testing.T) {
	sr := &SQLiteRouter{DBPath: "/nonexistent/path/database.db", Query: "SELECT host, port FROM routes WHERE domain = ?"}
	err := sr.Provision(caddy.Context{})
	if err == nil {
		t.Error("Expected Provision to fail with invalid database path, got nil")
	}
}

func TestCleanup(t *testing.T) {
	sr := &SQLiteRouter{DBPath: "test.db", Query: "SELECT host, port FROM route WHERE domain = ?"}
	if err := sr.Provision(caddy.Context{}); err != nil {
		t.Fatalf("Provision failed: %v", err)
	}
	if err := sr.Cleanup(); err != nil {
		t.Errorf("Cleanup failed: %v", err)
	}
	// Try to ping after cleanup - should fail with closed connection error
	err := sr.db.Ping()
	if err == nil {
		t.Error("Expected error when pinging closed database, got nil")
	}
}

func setupTest(t *testing.T, url string) (*SQLiteRouter, *http.Request, *httptest.ResponseRecorder) {
	sr := &SQLiteRouter{DBPath: "test.db", Query: "SELECT host, port FROM route WHERE domain = ?"}
	if err := sr.Provision(caddy.Context{}); err != nil { t.Fatalf("Provision failed: %v", err) }
	req := httptest.NewRequest("GET", url, nil)
	req = req.WithContext(context.WithValue(req.Context(), caddyhttp.VarsCtxKey, make(map[string]any)))
	return sr, req, httptest.NewRecorder()
}

func TestServeHTTP(t *testing.T) {
	sr, req, rec := setupTest(t, "http://app1.localhost/")
	nextCalled := false
	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		nextCalled = true
		upstream := caddyhttp.GetVar(r.Context(), "backend_upstream")
		if upstream != "localhost:8001" { t.Errorf("Expected upstream 'localhost:8001', got '%v'", upstream) }
		return nil
	})
	if err := sr.ServeHTTP(rec, req, next); err != nil { t.Errorf("ServeHTTP failed: %v", err) }
	if !nextCalled { t.Error("Expected next handler to be called") }
}

func TestServeHTTPNotFound(t *testing.T) {
	sr, req, rec := setupTest(t, "http://app3.localhost/")
	nextCalled := false
	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error { nextCalled = true; return nil })
	if err := sr.ServeHTTP(rec, req, next); err != nil { t.Errorf("ServeHTTP failed: %v", err) }
	if nextCalled { t.Error("Expected next handler NOT to be called for 404") }
	if rec.Code != http.StatusNotFound { t.Errorf("Expected status 404, got %d", rec.Code) }
}

func TestServeHTTPDatabaseError(t *testing.T) {
	sr, req, rec := setupTest(t, "http://app1.localhost/")
	// Close the database to cause an error
	sr.db.Close()
	nextCalled := false
	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error { 
		nextCalled = true
		return nil 
	})
	if err := sr.ServeHTTP(rec, req, next); err != nil { 
		t.Errorf("ServeHTTP failed: %v", err) 
	}
	if nextCalled { 
		t.Error("Expected next handler NOT to be called for database error") 
	}
	if rec.Code != http.StatusBadGateway { 
		t.Errorf("Expected status 502 (Bad Gateway), got %d", rec.Code) 
	}
}
