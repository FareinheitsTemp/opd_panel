#!/usr/bin/env bash
set -euo pipefail

BINARY=${1:-./opd}
SERVICE_FILE=$(dirname "$0")/opd.service

echo "[opd] Installing opd..."

# Build if binary not provided
if [ ! -f "$BINARY" ]; then
  echo "[opd] Building..."
  cd "$(dirname "$0")/.."
  go build -o opd .
  BINARY=./opd
fi

# Copy binary
sudo cp "$BINARY" /usr/local/bin/opd
sudo chmod +x /usr/local/bin/opd
echo "[opd] Binary installed to /usr/local/bin/opd"

# Create user
if ! id -u opd &>/dev/null; then
  sudo useradd -r -s /sbin/nologin -d /var/lib/opd opd
  echo "[opd] Created system user 'opd'"
fi

# Create directories
sudo mkdir -p /var/lib/opd/servers
sudo chown -R opd:opd /var/lib/opd
echo "[opd] Created /var/lib/opd/servers"

# Install systemd service
sudo cp "$SERVICE_FILE" /etc/systemd/system/opd.service
sudo systemd-analyze verify /etc/systemd/system/opd.service 2>/dev/null || true
sudo systemctl daemon-reload
sudo systemctl enable opd
echo "[opd] systemd service installed and enabled"

echo ""
echo "✔ Done. Start with: sudo systemctl start opd"
echo "  Logs: journalctl -u opd -f"
echo "  Add a server: opd add myserver"
