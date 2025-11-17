# caddy-sqlite-router

A Caddy module that routes requests based on subdomain lookups in a SQLite database.

## Usage

Build Caddy with this module using `xcaddy`:

```bash
CGO_ENABLED=1 xcaddy build --with github.com/AnswerDotAI/caddy-sqlite-router
```

The module extracts the subdomain from incoming requests and queries your database. Your query must:
- Accept exactly one named parameter `:domain` (the subdomain)
- Return exactly two columns: host (string) and port (integer)

Example Database Schema:

```sql
CREATE TABLE route (
  domain TEXT PRIMARY KEY,
  host TEXT NOT NULL,
  port INTEGER NOT NULL
);

INSERT INTO route VALUES ('app1', 'localhost', 8001);
INSERT INTO route VALUES ('app2', 'localhost', 8002);
```

The module sets the `backend_upstream` variable which can be used by reverse_proxy.

Example Caddyfile:

```caddyfile
*.localhost:9090 {
    route {
        sqlite_router test.db "SELECT host, port FROM route WHERE domain = :domain"
        reverse_proxy {http.vars.backend_upstream}
    }
}
```

This will reverse proxy visits to `https://app1.localhost:9090` to `localhost:8001`.

## Testing

1. Create the test database by running `python mkdb.py` with a python virtual environment that has `fastlite` installed.
2. Run `CGO_ENABLED=1 go test -v ./...` to run the unit tests with the SQLite3 driver.
3. Run `python test_e2e.py` to run the end to end test.

## License

Apache-2.0
