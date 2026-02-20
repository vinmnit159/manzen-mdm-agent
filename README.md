# Manzen MDM Agent

A lightweight, open-source Go agent for macOS endpoints. It collects device security posture data every 15 minutes and reports it to the [Manzen ISMS backend](https://github.com/vinmnit159/isms-backend), where it drives automatic risk creation, compliance evidence, and the Computers dashboard.

**Live backend:** `https://ismsbackend.bitcoingames1346.com`  
**Live frontend:** `https://isms.bitcoingames1346.com`

---

## How it fits into the system

```
Mac Device
   │
   ▼
manzen-agent (this repo)
   │  collects posture via macOS CLI tools
   │  signs request with HMAC-SHA256
   ▼
POST /api/agent/checkin
   │
   ▼
isms-backend
   │  validates API key
   │  updates DeviceCompliance table
   │  stores raw payload in DeviceCheckin (audit log)
   │  runs auto-risk engine → creates/mitigates ISO 27001 risks
   ▼
Manzen Web UI (Computers page + Risk dashboard)
```

### Enrollment flow

```
Admin (Web UI)
   │  creates enrollment token (24h TTL)
   ▼
User runs install.sh --token <TOKEN>
   │  downloads binary
   │  calls POST /api/agent/enroll
   ▼
Backend validates token, creates Asset record, issues API key
   │
   ▼
Agent saves config (~/.manzen-agent/config.json)
   │  deviceId + apiKey stored locally
   ▼
LaunchDaemon installed — agent runs every 15 min forever
```

---

## Requirements

- macOS (arm64 or amd64)
- Go 1.22+ (only needed to build from source)
- `sudo` access for enrollment and LaunchDaemon install

---

## Install (pre-built binary)

Go to **Manzen → Integrations → Manzen MDM Agent**, create an enrollment token, and copy the install command. It looks like:

```bash
curl -fsSL https://raw.githubusercontent.com/vinmnit159/manzen-mdm-agent/main/scripts/install.sh \
  -o /tmp/manzen-install.sh
sudo bash /tmp/manzen-install.sh --token <YOUR_ONE_TIME_TOKEN>
```

The script:
1. Downloads the correct binary for your architecture from the latest GitHub release
2. Runs `manzen-agent enroll` to register the device and receive an API key
3. Installs a `LaunchDaemon` at `/Library/LaunchDaemons/com.manzen.agent.plist` so the agent runs automatically on boot

Logs are written to `/var/log/manzen-agent.log`.

---

## Build from source

```bash
git clone git@github.com:vinmnit159/manzen-mdm-agent.git
cd manzen-mdm-agent

# Build for your current machine
go build -o manzen-agent ./cmd/agent

# Cross-compile for Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o manzen-agent-arm64 ./cmd/agent

# Cross-compile for Intel Mac
GOOS=darwin GOARCH=amd64 go build -o manzen-agent-amd64 ./cmd/agent
```

---

## CLI reference

```
manzen-agent enroll  --token <TOKEN> [--server <URL>]
manzen-agent checkin [--server <URL>]
manzen-agent daemon  [--server <URL>] [--interval 15m]
manzen-agent version
```

| Command | Description |
|---|---|
| `enroll` | One-time device registration using an admin-issued token. Saves `config.json`. |
| `checkin` | Collect posture and send one check-in immediately. |
| `daemon` | Loop: collect + check in on the given interval (default 15 min). |
| `version` | Print the agent version. |

---

## What is collected

Each check maps to an ISO 27001 Annex A control. Non-compliant checks automatically create risks in the ISMS.

| Check | macOS command used | ISO control | Risk if failing |
|---|---|---|---|
| Disk encryption (FileVault) | `fdesetup status` | A.8.24 | HIGH impact |
| Screen lock password | `defaults read com.apple.screensaver` | A.5.15 | MEDIUM impact |
| Host firewall | `socketfilterfw --getglobalstate` | A.8.20 | MEDIUM impact |
| System Integrity Protection | `csrutil status` | A.8.7 | HIGH impact |
| Automatic OS updates | `softwareupdate --schedule` | A.8.8 | MEDIUM impact |
| Gatekeeper | `spctl --status` | A.8.7 | MEDIUM impact |
| Antivirus (3rd party) | `ps aux` (scans for known AV processes) | A.8.7 | LOW impact |

---

## Project structure

```
manzen-mdm-agent/
├── cmd/
│   └── agent/
│       └── main.go              # CLI entry point — parses subcommands
├── internal/
│   ├── collector/
│   │   └── collector.go         # Runs macOS CLI tools, returns Posture struct
│   ├── enroll/
│   │   └── enroll.go            # POST /api/agent/enroll, saves config.json
│   └── checkin/
│       └── checkin.go           # HMAC-SHA256 signed POST /api/agent/checkin
├── scripts/
│   └── install.sh               # curl-pipe installer + LaunchDaemon setup
├── .github/
│   └── workflows/
│       └── release.yml          # GitHub Actions: builds binaries on every v* tag
├── go.mod
└── README.md
```

### Key files explained

**`internal/collector/collector.go`**  
Runs each macOS CLI tool in a subprocess, parses stdout, and assembles a `Posture` struct. All fields are `bool`. Fails gracefully — if a command is unavailable the field defaults to `false`.

**`internal/enroll/enroll.go`**  
Sends hostname, OS version, and serial number plus the one-time token to `POST /api/agent/enroll`. On success, receives a `deviceId` and `apiKey` which are saved to `~/.manzen-agent/config.json` (or `/Library/Application Support/ManzenAgent/config.json` when running as root).

**`internal/checkin/checkin.go`**  
Loads the saved config, collects posture, and POSTs to `/api/agent/checkin`. The request is authenticated with:
- `Authorization: Bearer <apiKey>` header
- `X-Device-ID: <deviceId>` header  
- `X-Agent-Signature` HMAC-SHA256 of `"<deviceId>:<unix_timestamp>"` using the API key as the secret

**`scripts/install.sh`**  
Detects architecture, downloads the matching binary from the latest GitHub release, runs enrollment, and writes a `launchctl` plist.

---

## Releases and CI

Every push of a `v*` tag triggers `.github/workflows/release.yml` which:
1. Cross-compiles three binaries: `darwin-arm64`, `darwin-amd64`, `linux-amd64`
2. Creates a GitHub Release and attaches all three as assets

To publish a new release:

```bash
git tag v0.2.0
git push origin v0.2.0
```

---

## Adding a new posture check

1. Add a function in `internal/collector/collector.go` that returns `bool`
2. Add the field to the `Posture` struct
3. Call it in `Collect()`
4. Add a matching rule to `COMPLIANCE_RULES` in `isms-backend/src/modules/agent/routes.ts` with the ISO reference, risk title, impact, and likelihood
5. Tag a new release

---

## Local development / testing without enrollment

You can test the collector in isolation without a real backend:

```bash
go run ./cmd/agent checkin --server http://localhost:3000
```

If you want to inspect what the collector returns without sending it anywhere, add a quick `main` to `internal/collector/collector.go` temporarily or write a small test:

```go
posture, _ := collector.Collect()
fmt.Printf("%+v\n", posture)
```
