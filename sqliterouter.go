package sqliterouter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(SQLiteRouter{})
	httpcaddyfile.RegisterHandlerDirective("sqlite_router", parseCaddyfile)
}

type SQLiteRouter struct {
	DBPath string `json:"db_path,omitempty"`
	Query  string `json:"query,omitempty"`
	db     *sql.DB
	stmt   *sql.Stmt
	logger *zap.Logger
}

func (SQLiteRouter) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{ID: "http.handlers.sqlite_router", New: func() caddy.Module { return new(SQLiteRouter) }}
}

func (m *SQLiteRouter) Provision(ctx caddy.Context) error {
	m.logger = ctx.Logger(m)
	var err error
	dsn := fmt.Sprintf("file:%s?mode=ro&_busy_timeout=3000", m.DBPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return err
	}

	closeDB := true
	defer func() {
		if closeDB {
			db.Close()
		}
	}()

	maxConns := runtime.NumCPU()
	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxConns)

	if err := db.PingContext(context.Background()); err != nil {
		return err
	}
	stmt, err := db.PrepareContext(context.Background(), m.Query)
	if err != nil {
		return err
	}

	closeStmt := true
	defer func() {
		if closeStmt {
			stmt.Close()
		}
	}()

	m.db = db
	m.stmt = stmt
	closeStmt = false
	closeDB = false
	return nil
}

func (m *SQLiteRouter) Cleanup() error {
	var err error
	if m.stmt != nil {
		if cerr := m.stmt.Close(); cerr != nil {
			err = cerr
			if m.logger != nil {
				m.logger.Error("failed to close prepared statement", zap.Error(cerr))
			}
		}
	}
	if m.db != nil {
		if cerr := m.db.Close(); cerr != nil {
			if err == nil {
				err = cerr
			}
			if m.logger != nil {
				m.logger.Error("failed to close database", zap.Error(cerr))
			}
		}
	}
	return err
}

func (m *SQLiteRouter) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if !d.Args(&m.DBPath, &m.Query) {
			return d.ArgErr()
		}
	}
	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	sr := new(SQLiteRouter)
	err := sr.UnmarshalCaddyfile(h.Dispenser)
	return sr, err
}

func (m SQLiteRouter) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	hostHeader := r.Host
	if h, _, err := net.SplitHostPort(hostHeader); err == nil {
		hostHeader = h
	}
	hostHeader = strings.TrimSuffix(strings.ToLower(hostHeader), ".")
	labels := strings.Split(hostHeader, ".")
	if len(labels) < 2 || labels[0] == "" {
		http.Error(w, "Invalid host", http.StatusBadRequest)
		if m.logger != nil {
			m.logger.Warn("invalid host header", zap.String("host", r.Host))
		}
		return nil
	}
	subdomain := labels[0]
	var host string
	var port int
	if err := m.stmt.QueryRowContext(r.Context(), sql.Named("domain", subdomain)).Scan(&host, &port); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusBadGateway)
			m.logger.Error("database query failed", zap.Error(err), zap.String("subdomain", subdomain))
		}
		return nil
	}
	upstream := fmt.Sprintf("%s:%d", host, port)
	caddyhttp.SetVar(r.Context(), "backend_upstream", upstream)
	return next.ServeHTTP(w, r)
}
