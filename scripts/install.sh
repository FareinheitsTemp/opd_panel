#!/usr/bin/env bash
set -euo pipefail

echo "==> OPD Panel installer"

# Generate agent secret
AGENT_SECRET=$(openssl rand -hex 32)

# Create directories
sudo mkdir -p /var/lib/opd/servers
sudo mkdir -p /var/lib/opd/cache
sudo mkdir -p /var/lib/opd/backups
sudo mkdir -p /var/run/opd
sudo mkdir -p /etc/opd

# Create opd group
if ! getent group opd > /dev/null 2>&1; then
  sudo groupadd opd
  echo "==> Created group 'opd'"
fi

# Write .env
if [ ! -f /etc/opd/.env ]; then
  cat > /tmp/opd.env << EOF
OPD_SOCKET=/var/run/opd/opd.sock
OPD_DB_PATH=/var/lib/opd/opd.db
OPD_AGENT_URL=http://127.0.0.1:7070
OPD_AGENT_SECRET=${AGENT_SECRET}
OPD_SERVERS_DIR=/var/lib/opd/servers
OPD_CACHE_DIR=/var/lib/opd/cache
OPD_LOG_LEVEL=info
EOF
  sudo mv /tmp/opd.env /etc/opd/.env
  sudo chmod 600 /etc/opd/.env
  echo "==> Generated /etc/opd/.env with random agent secret"
fi

# Build Go binaries
echo "==> Building panel & CLI..."
(cd panel && go build -o /usr/local/bin/opdd ./cmd/opdd && go build -o /usr/local/bin/opd ./cmd/opd)

# Build Rust agent
echo "==> Building agent..."
(cd agent && cargo build --release && sudo cp target/release/opd-agent /usr/local/bin/opd-agent)

# Install systemd units
sudo cp scripts/systemd/opdd.service /etc/systemd/system/
sudo cp scripts/systemd/opd-agent.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable opdd opd-agent

echo ""
echo "✓ OPD Panel installed"
echo "  Start:  sudo systemctl start opdd opd-agent"
echo "  Status: opd server list"
