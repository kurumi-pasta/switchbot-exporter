#!/bin/bash
# SwitchBot Exporter のデプロイスクリプト（Raspberry Pi Zero用）
set -e

# .envファイルからPI_USER_HOSTを読み込む
if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi

# Raspberry Pi Zero の情報
: "${PI_USER_HOST:?PI_USER_HOST is not set in .env}"
PI_BIN_PATH="/usr/local/bin/switchbot-exporter"
PI_SERVICE_PATH="/etc/systemd/system/switchbot-exporter.service"

# 1. GoバイナリをPi Zero用にクロスビルド
# Pi Zeroはarmv6/armhf (32bit) なので GOARCH=arm GOARM=6
GOOS=linux GOARCH=arm GOARM=6 go build -o switchbot-exporter .
echo "--- ビルド完了 ---"

# 2. Pi上の既存プロセスを停止
echo "--- Raspberry Pi上の既存プロセスを停止 ---"
ssh $PI_USER_HOST "sudo systemctl stop switchbot-exporter.service || true"

# 3. バイナリを一時ディレクトリに転送し、所定の場所へ移動
echo "--- バイナリをRaspberry Piに転送 ---"
scp switchbot-exporter "$PI_USER_HOST:/tmp/switchbot-exporter"
ssh $PI_USER_HOST "sudo mv /tmp/switchbot-exporter $PI_BIN_PATH && sudo chown root:root $PI_BIN_PATH"

# 4. サービスファイルを一時ディレクトリに転送し、所定の場所へ移動
echo "--- サービスファイルをRaspberry Piに転送 ---"
scp switchbot-exporter.service "$PI_USER_HOST:/tmp/switchbot-exporter.service"
ssh $PI_USER_HOST "sudo mv /tmp/switchbot-exporter.service $PI_SERVICE_PATH && sudo chown root:root $PI_SERVICE_PATH"

# 5. サービスのリロード・有効化・起動
echo "--- サービスのリロード・有効化・起動 ---"
ssh $PI_USER_HOST "sudo systemctl daemon-reload && sudo systemctl enable switchbot-exporter.service && sudo systemctl restart switchbot-exporter.service"

echo "--- デプロイ完了 ---"

# ビルドしたバイナリを削除
rm switchbot-exporter
