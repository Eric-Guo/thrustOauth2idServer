# Thruster compatibility port

This application ports the runtime changes from Thruster's
`6d415ec8e03392cd74e4b50d2051ce3222ead294^..3ef851682cd2a031294b7a2ac5e012a8bace0cfd`
range (after v0.1.16, through v0.1.26 and the September 2, 2026 dependency update).
It continues to use Sponge's application lifecycle, Gin routes, TLS server,
proxykit fallback, Ristretto cache, and structured logger.

## Change mapping

| Thruster change | Sponge implementation and application wiring |
| --- | --- |
| BREACH jitter and optional compression guard (`b1e717d`), compression middleware refactor (`b77cccb`, `6b05f45`) | `pkg/httpsrv` wraps the entire Gin engine, including native APIs, Rails fallback, cache hits, and X-Sendfile. The application's `http.gzipJitter` defaults to 32; `http.gzipDisableOnAuth` defaults to false. |
| Skip compressed image formats (`33ea926`) | The gzip content-type filter skips JPEG/JPG, PNG/APNG, WebP, GIF, AVIF, HEIC/HEIF, and JXL, including case variants and MIME parameters. SVG, BMP, TIFF, and text remain eligible. |
| Signal exit codes (`6898eea`) | `app.UpstreamServer` records shell-compatible exit codes. `app.RunWithExitCode` shuts down all services when the child exits, relays parent termination signals, and returns the child's status after cleanup. The application's main passes that status to `os.Exit`. |
| Restrict TLS redirects (`b090a2e`, `d925847`, `186b8f5`) | `httpsrv` normalizes hostnames with IDNA and checks the certificate manager's host policy. Rejected hosts get 421 without a Location header. Allowed hosts get 301, retaining the path/query and Sponge's configurable HTTPS port. ACME challenge handling remains outside the redirect handler. |
| Structured cache keys (`5a6ae0f`) | `proxykit/cache.CacheKey` separates method, host, escaped path, normalized query, and Vary headers. Ristretto remains the storage engine; entries retain their full key and reject hash collisions. Key bytes count toward the cache budget, and keys above 8 KiB are not stored. |
| Bypass uncacheable requests (`9b084d4`) | Method, Range, Upgrade, and oversized-key checks run before cache lookup. These requests bypass even a warm cache. Sponge's existing Vary lookup index is retained, bounded, and expired. |
| Request IDs (`e95cad7`, `f25e0fb`) | `pkg/requestid` assigns UUIDs before Gin. `proxy.forwardHeaders` controls whether client IDs are trusted. IDs reach Rails, responses, and access/proxy/cache logs; logged IDs are capped at 255 bytes. Proxy rewriting restores owned ID/start headers after hop-by-hop stripping, rejects upstream ID overrides, and avoids replaying IDs from cache. |
| Go/dependency updates, vet and vulnerability checks | Go 1.27.1 was already configured. Compression, x/crypto, and x/net now match the range endpoint: v1.20.0, v0.55.0, and v0.58.0. `make check` runs vet and the pinned govulncheck tool. |

Thruster's Ruby gem versions, release packaging, GitHub Actions changes, and
mechanical Go refactors have no application runtime equivalent to transplant.
They do not introduce additional Rails/proxy features. This port does not change
the application's existing Sponge CLI/configuration contract into a Ruby gem.

## Configuration

```yaml
http:
  gzipEnabled: true
  gzipJitter: 32
  gzipDisableOnAuth: false
  addRequestStartHeader: true
  logRequests: true
proxy:
  enabled: true
  forwardHeaders: false  # use false at the public TLS edge; true behind a trusted proxy
```

The guard disables locally applied gzip for requests with Cookie, Authorization,
or X-Csrf-Token, and responses with Set-Cookie, Cache-Control private/no-store, or
Vary: Cookie. The response guard runs before gzip chooses an encoding, including
explicit WriteHeader and Flush calls. It does not decode responses already
compressed by Rails; avoid compressing sensitive responses in the upstream when
using this control. Jitter applies to eligible compressed responses; zero
disables padding.

The following environment overrides retain Thruster's names:

| Environment variable | YAML setting |
| --- | --- |
| `GZIP_COMPRESSION_ENABLED` | `http.gzipEnabled` |
| `GZIP_COMPRESSION_DISABLE_ON_AUTH` | `http.gzipDisableOnAuth` |
| `GZIP_COMPRESSION_JITTER` | `http.gzipJitter` |
| `FORWARD_HEADERS` | `proxy.forwardHeaders` |

Each also accepts a `THRUSTER_` prefix. Prefixed values take precedence over
unprefixed values, and environment values override YAML. Invalid values fail
startup. Existing YAML files that omit gzipJitter receive the default of 32.
The sample's explicit `proxy.forwardHeaders: true` is preserved; change it to
false when this process terminates public TLS.

Native APIs remain in `internal/routers`, `internal/handler`, `internal/dao`, and
`internal/model`. A registered Sponge route takes precedence over the Rails
fallback even when its handler returns an error. Add critical APIs there while
Rails continues to serve unmatched routes.

## Building and validation

The implementation spans this application and the local Sponge checkout.
The existing `go.mod` replacement selects `/Users/guochunzhong/git/oss/sponge`;
both sets of changes are required. A release must use a Sponge revision containing
these changes, or an equivalent local replacement. Sponge v1.16.1 alone does not
contain the new APIs.

```sh
make test
go test -count=1 ./cmd/thrustOauth2idServer/...
make check
make ci-lint
go build -o /tmp/thrustOauth2idServer ./cmd/thrustOauth2idServer
```

In Sponge, the focused suite is:

```sh
go test -race -count=1 ./pkg/app ./pkg/httpsrv ./pkg/proxykit/cache ./pkg/gin/proxy ./pkg/requestid
```

The application integration test uses a temporary SQLite database and local
Rails stand-in, without production database or cache services. Process tests
start the actual main function in a subprocess and check clean exit, failure,
child signal death, and parent signal forwarding. TLS host-policy tests do not
request real certificates.

The application also pins OpenTelemetry API/SDK v1.42.0 to resolve the reachable
SDK and baggage-extraction findings exposed by the new vulnerability check.
Small existing lint/vet issues in the generated user code and tests were fixed
without changing API behavior.

Validated on September 9, 2026: the application suite, subprocess exit tests,
focused race tests, lint, vet, and binary build passed. Govulncheck found no
reachable vulnerabilities; it still reports findings in unused dependency code.
A fresh SQLite HTTP service generated from the modified Sponge templates also
compiled, with both compression settings present in its YAML and Go configuration.
