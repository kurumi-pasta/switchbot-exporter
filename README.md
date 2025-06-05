# SwitchBot Exporter

SwitchBot 温湿度計などのBLEデバイス情報をPrometheus形式でエクスポートするGoアプリケーションです。

## 特長
- SwitchBot温湿度計の温度・湿度・RSSIをPrometheusメトリクスとしてエクスポート
- デバイス情報（MACアドレス・名前）は `devices.json` で管理
- BLEスキャンは go-ble/ble を利用

## 使い方

### 1. 必要なファイルの準備

#### devices.json の例
```json
[
  { "address": "AA:BB:CC:DD:EE:01", "name": "温湿度計Pro" }
]
```

> **注意:** `devices.json` には自分のSwitchBotデバイスのMACアドレスと任意の名前を記載してください。

### 2. ビルド

```sh
go build -o switchbot-exporter main.go
```

### 3. 実行

```sh
./switchbot-exporter -p 8000
```

- `-p` オプションで exporter のポート番号を指定できます（デフォルト: 8000）

### 4. Prometheusメトリクスの確認

ブラウザで `http://localhost:8000/metrics` にアクセスすると、以下のようなメトリクスが出力されます。

```
switchbot_temperature_celsius{address="AA:BB:CC:DD:EE:01",name="温湿度計Pro"} 25.3
switchbot_humidity_percent{address="AA:BB:CC:DD:EE:01",name="温湿度計Pro"} 45
switchbot_rssi{address="AA:BB:CC:DD:EE:01",name="温湿度計Pro"} -60
```

### .envファイルについて

Raspberry Piの接続先などの環境依存情報は `.env` ファイルで管理できます。

#### .env の例
```
PI_USER_HOST="pi@raspberrypi.local"
```

- `dev_run_on_raspi.sh` や `deploy.sh` は `.env` から `PI_USER_HOST` を自動で読み込みます。

## サービスとしての運用例

Raspberry Pi等でsystemdサービスとして運用する場合は `switchbot-exporter.service` を参考にしてください。
