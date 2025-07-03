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
	c.readResponse()
	return nil
}

// CO2濃度を取得
func (c *MHZ19CClient) ReadCO2() (int, error) {
	cmd := []byte{0xFF, 0x01, 0x86, 0x00, 0x00, 0x00, 0x00, 0x00, 0x79}
	if _, err := c.p.Write(cmd); err != nil {
		return 0, err
	}

	buf, err := c.readResponse()
	if err != nil {
		return 0, err
	}
	if buf[1] != 0x86 {
		return 0, fmt.Errorf("unexpected format % X", buf)
	}

	co2 := int(buf[2])<<8 + int(buf[3])
	return co2, nil
}

func (c *MHZ19CClient) readResponse() ([]byte, error) {
	size := 9
	buf := make([]byte, size)
	readSize := 0
	retryCount := 0
	time.Sleep(time.Millisecond)
	for {
		retryCount++
		n, err := c.p.Read(buf[readSize:])
		if err != nil {
			return nil, err
		}

		readSize += n
		if readSize == size {
			break
		}

		if retryCount >= 20 {
			return nil, fmt.Errorf("timeout reading response, read %d bytes", readSize)
		}

		time.Sleep(time.Millisecond)
	}

	if buf[0] != 0xFF {
		return nil, fmt.Errorf("unexpected response start byte %02X, expected 0xFF", buf[0])
	}

	return buf, nil
}
