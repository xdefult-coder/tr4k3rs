#!/bin/bash

echo ""
echo "[*] Kali Location Tracker Installer"
echo "--------------------------------------"
echo ""

INSTALL_DIR="/opt/kali-location"
BIN_SERVER="/usr/local/bin/kali-location-server"
BIN_CLIENT="/usr/local/bin/kali-location-client"
SERVICE_FILE="/etc/systemd/system/kali-location.service"

echo "[*] Moving into project directory..."
cd "$INSTALL_DIR" || { echo "[!] Folder not found: $INSTALL_DIR"; exit 1; }

echo "[*] Checking Go module..."
if [ ! -f go.mod ]; then
    echo "[*] go.mod not found — creating..."
    go mod init kali-location
    go mod tidy
else
    echo "[*] go.mod found — running tidy..."
    go mod tidy
fi

echo "[*] Building Go binaries..."
go build -o "$BIN_SERVER" server.go || { echo "[!] Build failed (server.go)"; exit 1; }
go build -o "$BIN_CLIENT" client.go || { echo "[!] Build failed (client.go)"; exit 1; }

chmod +x "$BIN_SERVER"
chmod +x "$BIN_CLIENT"

echo "[*] Installing Systemd service..."
cat <<EOF > "$SERVICE_FILE"
[Unit]
Description=Kali Location Tracker
After=network.target

[Service]
Type=simple
ExecStart=$BIN_SERVER
WorkingDirectory=$INSTALL_DIR
Restart=always
RestartSec=5
User=root

[Install]
WantedBy=multi-user.target
EOF

echo "[*] Reloading systemd..."
systemctl daemon-reload

echo "[*] Enabling service..."
systemctl enable kali-location.service

echo "[*] Starting service..."
systemctl start kali-location.service

echo ""
echo "[✓] Installation complete!"
echo "[✓] Server running at: http://localhost:8080"
echo ""
echo "Use this command to check logs:"
echo "    journalctl -u kali-location -f"
echo ""
