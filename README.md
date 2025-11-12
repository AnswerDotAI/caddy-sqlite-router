# caddy-sqlite-router

A Caddy module that routes requests based on subdomain lookups in a SQLite database.

## Installation

Build Caddy with this module using `xcaddy`:

```bash
xcaddy build --with github.com/AnswerDotAI/caddy-sqlite-router
```

## Configuration

### JSON API

Example: 

```json
{
  "handler": "sqlite_router",
  "db_path": "/path/to/database.db",
  "query": "SELECT host, port FROM routes WHERE domain = ?"
}
```

### Caddyfile

```
sqlite_router /path/to/database.db "SELECT host, port FROM routes WHERE domain = ?"
```

## Usage

The module extracts the subdomain from incoming requests and queries your database. Your query must:
- Accept exactly one parameter (the subdomain)
- Return exactly two columns: host (string) and port (integer)

The module sets the `backend_upstream` variable which can be used by reverse_proxy:

```json
{
  "handler": "reverse_proxy",
  "upstreams": [{"dial": "{http.vars.backend_upstream}"}]
}
```

## Example Database Schema

```sql
CREATE TABLE routes (
  domain TEXT PRIMARY KEY,
  host TEXT NOT NULL,
  port INTEGER NOT NULL
);

INSERT INTO routes VALUES ('app1', 'localhost', 8001);
INSERT INTO routes VALUES ('app2', 'localhost', 8002);
```

## License

MIT
