# heimdall-cli

Admin CLI for Heimdall Server installations on Linux. Handles install-time setup (PostgreSQL bootstrap, fapolicyd trust, firewalld, SELinux), day-2 admin (status, reconfigure, backup), and is invoked from `heimdall-server` RPM scriptlets.

## Versioning

heimdall-cli uses **independent semver**, decoupled from `heimdall-server`. The CLI and the server ship on their own cadence; compatibility is expressed via [`COMPAT.md`](./COMPAT.md) (cli → server version range) and via RPM `Requires:` ranges in `heimdall-server.spec`, not lockstep version pins.

This follows the pattern of `kubectl`, `gh`, `stripe-cli`, `heroku-cli` — standalone CLIs that manage a service.

| Version | Meaning |
|---|---|
| `0.9.x` | Pre-1.0 stabilization. Real and shipping; API and UX may still change. |
| `1.0.0` | Reserved for after the TUI/admin-UX improvements land and we have real-world soak time. Commits to backwards compat. |

`heimdall-cli --version` on an unbuilt/dev binary reports `0.9.0-dev`. Released binaries report the git tag (e.g. `0.9.0`).

## Build

```sh
make build           # native binary
make build-fips      # FIPS-mode binary (boringcrypto)
make snapshot        # full goreleaser snapshot under dist/
```

## Tests

See [`docs/testing.md`](./docs/testing.md) for the unit / functional / integration / e2e pyramid.
