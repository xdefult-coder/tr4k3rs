#!/bin/bash
set -e
REPO_DIR="/opt/kali-location"   # where your git repo is checked out
SERVICE="kali-location.service"

cd "$REPO_DIR"
echo "[*] Pulling latest changes..."
git pull origin main

echo "[*] Rebuilding..."
go build -o /tmp/kali-location-server server.go

echo "[*] Installing new binary..."
sudo install -m 0755 /tmp/kali-location-server /usr/local/bin/kali-location-server

echo "[*] Restarting service..."
sudo systemctl restart $SERVICE
echo "[+] Update complete."
