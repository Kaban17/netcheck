package domain

import (
	"context"
	"time"
)

type NetworkRepository interface {
	TCPPing(ctx context.Context, addr string, timeout time.Duration) (time.Duration, error)
	LookupHost(ctx context.Context, host string) ([]string, time.Duration, error)
	Download(ctx context.Context, url string) (int64, time.Duration, error)
	GetPublicIP(ctx context.Context) (IPInfo, error)
	GetLocalInterfaces() ([]InterfaceInfo, error)
	DetectGateway() string
}
