#!/bin/bash
set -e
APP_NAME="kali-location-server"
BUILD_OUT="./server_bin"
echo "[*] Building Go binaries..."
go mod tidy
go build -o ${BUILD_OUT} server.go

echo "[*] Installing binary to /usr/local/bin/${APP_NAME}"
sudo install -m 0755 ${BUILD_OUT} /usr/local/bin/${APP_NAME}

echo "[*] Installing viewer and static files to /opt/kali-location"
sudo mkdir -p /opt/kali-location
sudo cp -r viewer.html static /opt/kali-location/
sudo chown -R root:root /opt/kali-location

# systemd unit
SERVICE_FILE="/etc/systemd/system/kali-location.service"
echo "[*] Creating systemd service: ${SERVICE_FILE}"
sudo tee ${SERVICE_FILE} > /dev/null <<'EOF'
[Unit]
Description=Kali Location Tracker
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/kali-location-server
WorkingDirectory=/opt/kali-location
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
EOF

echo "[*] Reloading systemd and enabling service..."
sudo systemctl daemon-reload
sudo systemctl enable kali-location.service
sudo systemctl start kali-location.service
echo "[+] Installation complete. Service started."
