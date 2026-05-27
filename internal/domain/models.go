package domain

import "time"

type PingResult struct {
	Name    string
	Host    string
	Samples []time.Duration
	Errors  int
}

type DNSResult struct {
	Host    string
	Elapsed time.Duration
	IPs     []string
	Success bool
}

type SpeedResult struct {
	Label    string
	Bytes    int64
	Elapsed  time.Duration
	MbPerSec float64
}

type IPInfo struct {
	IP      string `json:"ip"`
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
	Org     string `json:"org"`
}

type InterfaceInfo struct {
	Name  string
	IPs   []string
	Flags []string
}

type NetworkInfo struct {
	PublicIP IPInfo
	Interfaces []InterfaceInfo
	Gateway    string
	GatewayRTT time.Duration
}
