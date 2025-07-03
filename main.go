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
	"time"

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

// 温湿度計
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

// MH-Z19C
var (
	co2Gauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "co2_ppm",
			Help: "CO2濃度",
		},
	)
)

func init() {
	prometheus.MustRegister(temperatureGauge)
	prometheus.MustRegister(humidityGauge)
	prometheus.MustRegister(rssiGauge)
	prometheus.MustRegister(co2Gauge)
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

	go func() {
		mhz19cClient, err := Open("/dev/serial0")
		if err != nil {
			slog.Error("MH-Z19Cへの接続失敗", "error", err)
			return
		}

		mhz19cClient.DisableABC()

		readTicker := time.NewTicker(10 * time.Second)
		abcTicker := time.NewTicker(1 * time.Hour)
		for {
			select {
			case <-context.Background().Done():
				return
			case <-readTicker.C:
				co2, err := mhz19cClient.ReadCO2()
				if err != nil {
					slog.Error("CO2濃度取得失敗", "error", err)
					co2Gauge.Set(math.NaN())
				} else {
					slog.Info("CO2濃度取得", "co2", co2)
					co2Gauge.Set(float64(co2))
				}
			case <-abcTicker.C:
				if err := mhz19cClient.DisableABC(); err != nil {
					slog.Error("自動校正の無効化に失敗", "error", err)
				} else {
					slog.Info("自動校正を無効化しました")
				}
			}
		}
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
		slog.Error("スキャンエラー", "error", err)
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
