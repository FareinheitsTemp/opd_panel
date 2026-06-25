#!/usr/bin/env bash
set -euo pipefail

echo "[opd] Stopping service..."
sudo systemctl stop opd 2>/dev/null || true
sudo systemctl disable opd 2>/dev/null || true
sudo rm -f /etc/systemd/system/opd.service
sudo systemctl daemon-reload

echo "[opd] Removing binary..."
sudo rm -f /usr/local/bin/opd

echo "[opd] Note: /var/lib/opd (server data) was NOT removed."
echo "  To remove it manually: sudo rm -rf /var/lib/opd"
echo ""
echo "✔ opd uninstalled."
