# skyip (fork)

Usage:

- Build (simple):

```bash
go build -o skyip ./cmd/skyip
```

- Build (with version info set at build time):

```bash
# Example: set version, commit and build date
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=1.2.3 -X main.commit=$(git rev-parse --short HEAD) -X 'main.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" -o skyip ./cmd/skyip
```

- Show version info at runtime:

```bash
./skyip -v
# or
./skyip --version
```

- Run:

```bash
./skyip run -f /path/to/config.json [-i]
```

Proxy configuration

- The `proxy` field in `config.json` must include a scheme. Supported schemes:
  - `http://host:port` or `https://host:port`
  - `socks5://host:port` or `socks5h://host:port`

Example `config.json` snippet:

```json
{
  "proxy": "socks5://127.0.0.1:1080"
}
```

Notes:
- If you set a SOCKS5 proxy the program will use it for all Cloudflare API requests.
- If `proxy` is empty or omitted, direct connections are used.

## Project layout

The repository has been reorganized into a small, opinionated structure:

- `cmd/skyip/` - the `main` package and command-line entrypoint
- `internal/config/` - configuration parsing and cache helpers
- `internal/netlink/` - Linux netlink interaction and IPv6 selection logic
- `internal/provider/cloudflare/` - Cloudflare API implementation
- `internal/log/` - logging helpers

I kept the original top-level sources in place (not deleted) as a backup. If you'd like, I can remove or archive the old files — tell me and I'll do that after your confirmation.
