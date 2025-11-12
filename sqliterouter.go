package sqliterouter

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
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
}

func (SQLiteRouter) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{ID: "http.handlers.sqlite_router", New: func() caddy.Module { return new(SQLiteRouter) }}
}

func (m *SQLiteRouter) Provision(ctx caddy.Context) error {
	var err error
	m.db, err = sql.Open("sqlite", m.DBPath)
	return err
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
	if err := m.db.QueryRow(m.Query, subdomain).Scan(&host, &port); err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return nil
	}
	upstream := fmt.Sprintf("%s:%d", host, port)
	caddyhttp.SetVar(r.Context(), "backend_upstream", upstream)
	return next.ServeHTTP(w, r)
}
