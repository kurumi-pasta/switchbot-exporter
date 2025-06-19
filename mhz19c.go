package main

import (
	"fmt"
	"time"

	"github.com/tarm/serial"
)

// https://github.com/UedaTakeyuki/mh-z19/blob/master/mh_z19.py

type MHZ19CClient struct {
	p *serial.Port
}

func Open(device string) (*MHZ19CClient, error) {
	c := &serial.Config{
		Name:        device,
		Baud:        9600,
		ReadTimeout: time.Second * 2,
	}
	p, err := serial.OpenPort(c)
	if err != nil {
		return nil, err
	}

	return &MHZ19CClient{p: p}, nil
}

func (c *MHZ19CClient) Close() error {
	return c.p.Close()
}

// 自動校正を無効にする
func (c *MHZ19CClient) DisableABC() error {
	cmd := []byte{0xFF, 0x01, 0x79, 0x00, 0x00, 0x00, 0x00, 0x00, 0x86}
	if _, err := c.p.Write(cmd); err != nil {
		return err
	}
	// レスポンスを消費
	time.Sleep(100 * time.Millisecond)
	buf := make([]byte, 9)
	c.p.Read(buf)

	return nil
}

// CO2濃度を取得
func (c *MHZ19CClient) ReadCO2() (int, error) {
	cmd := []byte{0xFF, 0x01, 0x86, 0x00, 0x00, 0x00, 0x00, 0x00, 0x79}
	if _, err := c.p.Write(cmd); err != nil {
		return 0, err
	}

	time.Sleep(100 * time.Millisecond)
	buf := make([]byte, 9)
	// 0xFFまで消費
	for {
		if _, err := c.p.Read(buf[0:1]); err != nil {
			return 0, err
		}
		if buf[0] == 0xFF {
			break
		}
	}

	// 残りを読み込む
	if _, err := c.p.Read(buf[1:]); err != nil {
		return 0, err
	}

	if buf[1] != 0x86 {
		return 0, fmt.Errorf("unexpected format % X", buf)
	}

	co2 := int(buf[2])<<8 + int(buf[3])
	return co2, nil
}
