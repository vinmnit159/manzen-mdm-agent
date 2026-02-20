#!/bin/bash
# Manzen MDM Agent — macOS install script
# Usage: sudo bash install.sh --token <ONE_TIME_TOKEN>
set -euo pipefail

AGENT_URL="https://github.com/vinmnit159/manzen-mdm-agent/releases/latest/download/manzen-agent-darwin-arm64"
INSTALL_PATH="/usr/local/bin/manzen-agent"
PLIST_PATH="/Library/LaunchDaemons/com.manzen.agent.plist"
SERVER_URL="https://ismsbackend.bitcoingames1346.com"
INTERVAL=900  # 15 minutes

TOKEN=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --token) TOKEN="$2"; shift 2 ;;
    --server) SERVER_URL="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [[ -z "$TOKEN" ]]; then
  echo "Usage: sudo bash install.sh --token <ENROLLMENT_TOKEN>"
  exit 1
fi

if [[ "$EUID" -ne 0 ]]; then
  echo "Please run as root (sudo)"
  exit 1
fi

echo "==> Downloading Manzen MDM Agent..."
curl -fsSL "$AGENT_URL" -o "$INSTALL_PATH"
chmod +x "$INSTALL_PATH"

echo "==> Enrolling device..."
"$INSTALL_PATH" enroll --token "$TOKEN" --server "$SERVER_URL"

echo "==> Installing LaunchDaemon..."
cat > "$PLIST_PATH" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.manzen.agent</string>
  <key>ProgramArguments</key>
  <array>
    <string>${INSTALL_PATH}</string>
    <string>daemon</string>
    <string>--server</string>
    <string>${SERVER_URL}</string>
    <string>--interval</string>
    <string>15m</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>/var/log/manzen-agent.log</string>
  <key>StandardErrorPath</key>
  <string>/var/log/manzen-agent.log</string>
</dict>
</plist>
EOF

launchctl load -w "$PLIST_PATH"

echo ""
echo "✅ Manzen MDM Agent installed and running."
echo "   Logs: /var/log/manzen-agent.log"
echo "   First check-in will appear in the ISMS dashboard within 1 minute."
