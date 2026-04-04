# ManZen MDM Agent

## Stack
- **Language**: Go 1.22
- **Entry point**: `cmd/agent/main.go`
- **Distribution**: GitHub Releases (binary builds)

## Key Commands
| Command | Purpose |
|---------|---------|
| `go vet ./...` | Static analysis |
| `go test ./...` | Run tests |
| `go build ./cmd/agent` | Build binary |

## CI Workflows
| Workflow | Trigger | Steps |
|----------|---------|-------|
| `ci.yml` | Push/PR to main | go vet, go test, go build |
| `release.yml` | Tag push (`v*`) | Build multi-platform binaries, create GitHub Release |

## Deploy Targets
| Environment | Platform | Trigger |
|-------------|----------|---------|
| Release | GitHub Releases | Push `v*` tag |

## Repo-Specific Rules
- Agent connects to `api.cloudanzen.com` — never hardcode old domains
- Install script at `scripts/install.sh` must reference correct API URL
- No staging environment for the agent itself — test against staging API manually
- Binary releases are multi-platform (Linux amd64/arm64, macOS, Windows)
- No test suite currently — add tests for any new functionality

## Versioning
- Current: `v1.0.0` (baseline set 2026-04-04)
- Versioned independently
- Release via `v*` git tags → GitHub Release workflow
