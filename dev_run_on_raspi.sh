#!/bin/bash
# SwitchBot Prometheus Exporter をRaspberry Piで開発・動作確認するためのスクリプト
set -e

# .envファイルからPI_USER_HOSTを読み込む
if [ -f .env ]; then
  set -a
  . ./.env
  set +a
fi

# Raspberry Pi の接続情報
# PI_USER_HOSTは.envから取得
: "${PI_USER_HOST:?PI_USER_HOST is not set in .env}"
PI_PATH="/home/pi/switchbot-exporter"

# 1. クロスビルド（Mac上でRaspberry Pi用バイナリをビルド）
echo "--- ビルド開始 ---"
GOOS=linux GOARCH=arm GOARM=6 go build -o switchbot-exporter .
echo "--- ビルド完了 ---"

# 2. バイナリをRaspberry Piに転送
echo "--- バイナリをRaspberry Piに転送 ---"
scp switchbot-exporter "$PI_USER_HOST:$PI_PATH"

# 3. 転送後にローカルのバイナリを削除
rm switchbot-exporter

echo "--- Raspberry Pi上で以下のコマンドを実行してください ---"
echo "sudo $PI_PATH"
