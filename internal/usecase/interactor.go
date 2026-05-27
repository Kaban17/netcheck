package usecase

import (
	"context"
	"time"

	"netcheck/internal/domain"
)

type NetworkUseCase interface {
	RunPing(ctx context.Context, hosts []struct{ Name, Addr string }, samples int, progress chan<- string) []domain.PingResult
	RunDNS(ctx context.Context, domains []string) []domain.DNSResult
	RunSpeed(ctx context.Context, tests []struct {
		Label string
		URL   string
	}) []domain.SpeedResult
	RunInfo(ctx context.Context) (domain.NetworkInfo, error)
}

type networkInteractor struct {
	repo domain.NetworkRepository
}

func NewNetworkInteractor(repo domain.NetworkRepository) NetworkUseCase {
	return &networkInteractor{repo: repo}
}

func (uc *networkInteractor) RunPing(ctx context.Context, hosts []struct{ Name, Addr string }, samples int, progress chan<- string) []domain.PingResult {
	results := make([]domain.PingResult, len(hosts))
	for i, h := range hosts {
		r := domain.PingResult{Name: h.Name, Host: h.Addr}
		if progress != nil {
			progress <- h.Name
		}
		for s := 0; s < samples; s++ {
			rtt, err := uc.repo.TCPPing(ctx, h.Addr, 3*time.Second)
			if err != nil {
				r.Errors++
			} else {
				r.Samples = append(r.Samples, rtt)
			}
			time.Sleep(150 * time.Millisecond)
		}
		results[i] = r
	}
	return results
}

func (uc *networkInteractor) RunDNS(ctx context.Context, domains []string) []domain.DNSResult {
	results := make([]domain.DNSResult, len(domains))
	for i, d := range domains {
		ips, elapsed, err := uc.repo.LookupHost(ctx, d)
		results[i] = domain.DNSResult{
			Host:    d,
			Elapsed: elapsed,
			IPs:     ips,
			Success: err == nil,
		}
	}
	return results
}

func (uc *networkInteractor) RunSpeed(ctx context.Context, tests []struct {
	Label string
	URL   string
}) []domain.SpeedResult {
	results := make([]domain.SpeedResult, len(tests))
	for i, t := range tests {
		bytes, elapsed, err := uc.repo.Download(ctx, t.URL)
		if err != nil {
			continue // Or handle error
		}
		mbps := float64(bytes) * 8 / elapsed.Seconds() / 1_000_000
		results[i] = domain.SpeedResult{
			Label:    t.Label,
			Bytes:    bytes,
			Elapsed:  elapsed,
			MbPerSec: mbps,
		}
	}
	return results
}

func (uc *networkInteractor) RunInfo(ctx context.Context) (domain.NetworkInfo, error) {
	info := domain.NetworkInfo{}
	
	publicIP, err := uc.repo.GetPublicIP(ctx)
	if err == nil {
		info.PublicIP = publicIP
	}

	ifaces, err := uc.repo.GetLocalInterfaces()
	if err == nil {
		info.Interfaces = ifaces
	}

	gw := uc.repo.DetectGateway()
	if gw != "" {
		info.Gateway = gw
		rtt, err := uc.repo.TCPPing(ctx, gw+":80", 1*time.Second)
		if err != nil {
			rtt, _ = uc.repo.TCPPing(ctx, gw+":443", 1*time.Second)
		}
		info.GatewayRTT = rtt
	}

	return info, nil
}
