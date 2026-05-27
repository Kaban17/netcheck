package network

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"netcheck/internal/domain"
)

type networkRepository struct {
	httpClient *http.Client
}

func NewNetworkRepository() domain.NetworkRepository {
	return &networkRepository{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (r *networkRepository) TCPPing(ctx context.Context, addr string, timeout time.Duration) (time.Duration, error) {
	d := net.Dialer{Timeout: timeout}
	start := time.Now()
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return time.Since(start), nil
}

func (r *networkRepository) LookupHost(ctx context.Context, host string) ([]string, time.Duration, error) {
	start := time.Now()
	ips, err := net.DefaultResolver.LookupHost(ctx, host)
	return ips, time.Since(start), err
}

func (r *networkRepository) Download(ctx context.Context, url string) (int64, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "netcheck/1.0")

	start := time.Now()
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	n, err := io.Copy(io.Discard, resp.Body)
	return n, time.Since(start), err
}

func (r *networkRepository) GetPublicIP(ctx context.Context) (domain.IPInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://ipinfo.io/json", nil)
	if err != nil {
		return domain.IPInfo{}, err
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return domain.IPInfo{}, err
	}
	defer resp.Body.Close()

	var info domain.IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return domain.IPInfo{}, err
	}
	return info, nil
}

func (r *networkRepository) GetLocalInterfaces() ([]domain.InterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var results []domain.InterfaceInfo
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		var ips []string
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				ips = append(ips, ipnet.String())
			}
		}
		if len(ips) == 0 {
			continue
		}

		flags := []string{}
		if iface.Flags&net.FlagBroadcast != 0 {
			flags = append(flags, "broadcast")
		}
		if iface.Flags&net.FlagMulticast != 0 {
			flags = append(flags, "multicast")
		}

		results = append(results, domain.InterfaceInfo{
			Name:  iface.Name,
			IPs:   ips,
			Flags: flags,
		})
	}
	return results, nil
}

func (r *networkRepository) DetectGateway() string {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return ""
	}
	defer conn.Close()
	addr := conn.LocalAddr().(*net.UDPAddr)
	ip := addr.IP.To4()
	if ip == nil {
		return ""
	}
	return fmt.Sprintf("%d.%d.%d.1", ip[0], ip[1], ip[2])
}
