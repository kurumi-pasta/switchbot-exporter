package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"log/slog"

	"github.com/go-ble/ble"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//go:embed devices.json
var devicesJson []byte

// デバイス情報構造体
// devices.json用
type Device struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

// MACアドレス→名前のマップを作る関数
func loadDevices() (map[string]string, error) {
	var devices []Device
	if err := json.Unmarshal(devicesJson, &devices); err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, d := range devices {
		m[strings.ToUpper(d.Address)] = d.Name
	}
	return m, nil
}

// 対象デバイスのMACアドレスと名前のマップ
var switchbotDevices = map[string]string{}

var (
	temperatureGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "switchbot_temperature_celsius",
			Help: "SwitchBotの温度 (摂氏)",
		},
		[]string{"address", "name"},
	)
	humidityGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "switchbot_humidity_percent",
			Help: "SwitchBotの湿度(%)",
		},
		[]string{"address", "name"},
	)
	rssiGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "switchbot_rssi",
			Help: "SwitchBotのRSSI",
		},
		[]string{"address", "name"},
	)
)

func init() {
	prometheus.MustRegister(temperatureGauge)
	prometheus.MustRegister(humidityGauge)
	prometheus.MustRegister(rssiGauge)
}

func main() {
	// ポート番号のコマンドラインオプション
	port := flag.Int("p", 8000, "Prometheus exporterのポート番号")
	flag.Parse()

	// Prometheus metrics HTTPサーバ起動
	go func() {
		addr := fmt.Sprintf(":%d", *port)
		http.Handle("/metrics", promhttp.Handler())
		slog.Info("Prometheus metrics exporter started", "url", fmt.Sprintf("http://localhost:%d/metrics", *port))
		http.ListenAndServe(addr, nil)
	}()

	// デバイス情報の読み込み
	devices, err := loadDevices()
	if err != nil {
		slog.Error("devices.jsonの読み込みに失敗", "error", err)
		os.Exit(1)
	}
	switchbotDevices = devices

	// BLEデバイス初期化
	initBleDevice()

	ctx := ble.WithSigHandler(signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM))
	slog.Info("スキャン開始")
	err = ble.Scan(ctx, false, advHandler, nil)
	if err != nil {
		slog.Error("scan error", "error", err)
		os.Exit(1)
	}
}

// アドバタイズパケット受信時の処理
func advHandler(a ble.Advertisement) {
	addr := strings.ToUpper(a.Addr().String())
	name, ok := switchbotDevices[addr]
	if !ok {
		slog.Debug("無視: 対象外デバイス", "address", addr, "local_name", a.LocalName())
		return // 対象外デバイスは無視
	}
	slog.Info("デバイス検出", "name", name, "address", addr)

	data := a.ManufacturerData()
	if len(data) >= 13 && (data[0] == 0x48 || data[0] == 0x69) && data[1] == 0x09 {
		temperature, humidity := parseSwitchBotData(data)
		labels := []string{addr, name}
		temperatureGauge.WithLabelValues(labels...).Set(temperature)
		humidityGauge.WithLabelValues(labels...).Set(float64(humidity))
		rssiGauge.WithLabelValues(labels...).Set(float64(a.RSSI()))
		slog.Info("データ取得", "温度", temperature, "湿度", humidity, "RSSI", a.RSSI())
	} else {
		slog.Warn("対象外のデータ", "data", data)
		// 対象外のデータはNaNに設定
		labels := []string{addr, name}
		temperatureGauge.WithLabelValues(labels...).Set(math.NaN())
		humidityGauge.WithLabelValues(labels...).Set(math.NaN())
		rssiGauge.WithLabelValues(labels...).Set(math.NaN())
	}
}

// SwitchBotの生データから温度・湿度をパース
func parseSwitchBotData(raw []byte) (float64, int) {
	isTemperatureAboveFreezing := (raw[11] & 0b10000000) != 0
	temperature := float64(raw[10]&0b00001111)/10.0 + float64(raw[11]&0b01111111)
	if !isTemperatureAboveFreezing {
		temperature = -temperature
	}
	humidity := int(raw[12] & 0b01111111)
	return temperature, humidity
}
