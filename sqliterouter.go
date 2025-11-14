package sqliterouter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

func init() {
	caddy.RegisterModule(SQLiteRouter{})
	httpcaddyfile.RegisterHandlerDirective("sqlite_router", parseCaddyfile)
}

type SQLiteRouter struct{
	DBPath string `json:"db_path,omitempty"`
	Query string `json:"query,omitempty"`
	db *sql.DB
	stmt *sql.Stmt
	logger *zap.Logger
}

func (SQLiteRouter) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{ID: "http.handlers.sqlite_router", New: func() caddy.Module { return new(SQLiteRouter) }}
}

func (m *SQLiteRouter) Provision(ctx caddy.Context) error {
	m.logger = ctx.Logger(m)
	var err error
	dsn := m.DBPath + "?mode=ro&_pragma=busy_timeout(3000)"
	m.db, err = sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	
	maxConns := runtime.NumCPU()
	m.db.SetMaxOpenConns(maxConns)
	m.db.SetMaxIdleConns(maxConns)
	
	if err := m.db.PingContext(context.Background()); err != nil {
		return err
	}
	m.stmt, err = m.db.PrepareContext(context.Background(), m.Query)
	return err
}

func (m *SQLiteRouter) Cleanup() error {
	if m.stmt != nil {
		m.stmt.Close()
	}
	return m.db.Close()
}

func (m *SQLiteRouter) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if !d.Args(&m.DBPath, &m.Query) { return d.ArgErr() }
	}
	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	sr := new(SQLiteRouter)
	err := sr.UnmarshalCaddyfile(h.Dispenser)
	return sr, err
}

func (m SQLiteRouter) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	subdomain := strings.Split(strings.Split(r.Host, ".")[0], ":")[0]
	var host string
	var port int
	if err := m.stmt.QueryRowContext(r.Context(), subdomain).Scan(&host, &port); err != nil {
		m.logger.Error("database query failed", zap.Error(err), zap.String("subdomain", subdomain))
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusBadGateway)
		}
		return nil
	}
	upstream := fmt.Sprintf("%s:%d", host, port)
	caddyhttp.SetVar(r.Context(), "backend_upstream", upstream)
	return next.ServeHTTP(w, r)
}
