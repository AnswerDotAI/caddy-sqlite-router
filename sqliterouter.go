package sqliterouter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
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
	logger *zap.Logger
}

func (SQLiteRouter) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{ID: "http.handlers.sqlite_router", New: func() caddy.Module { return new(SQLiteRouter) }}
}

func (m *SQLiteRouter) Provision(ctx caddy.Context) error {
	m.logger = ctx.Logger(m)
	var err error
	m.db, err = sql.Open("sqlite", m.DBPath)
	if err != nil {
		return err
	}
	return m.db.PingContext(context.Background())
}

func (m *SQLiteRouter) Cleanup() error {
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
	m.logger.Info("extracted subdomain", zap.String("subdomain", subdomain), zap.String("host", r.Host))
	var host string
	var port int
	if err := m.db.QueryRowContext(r.Context(), m.Query, subdomain).Scan(&host, &port); err != nil {
		m.logger.Error("database query failed", zap.Error(err), zap.String("subdomain", subdomain))
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusBadGateway)
		}
		return nil
	}
	upstream := fmt.Sprintf("%s:%d", host, port)
	m.logger.Info("routing request", zap.String("subdomain", subdomain), zap.String("upstream", upstream))
	caddyhttp.SetVar(r.Context(), "backend_upstream", upstream)
	return next.ServeHTTP(w, r)
}
