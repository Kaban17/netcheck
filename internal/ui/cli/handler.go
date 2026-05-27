package cli

import (
	"context"
	"fmt"
	"strings"

	"netcheck/internal/ui"
	"netcheck/internal/usecase"
)

type Handler struct {
	useCase   usecase.NetworkUseCase
	presenter *Presenter
}

func NewHandler(uc usecase.NetworkUseCase, p *Presenter) *Handler {
	return &Handler{
		useCase:   uc,
		presenter: p,
	}
}

func (h *Handler) Handle(ctx context.Context, args []string) {
	cmd := "all"
	if len(args) >= 1 {
		cmd = strings.ToLower(args[0])
	}

	h.presenter.PrintBanner()

	switch cmd {
	case "ping":
		h.runPing(ctx)
	case "dns":
		h.runDNS(ctx)
	case "speed":
		h.runSpeed(ctx)
	case "info":
		h.runInfo(ctx)
	case "all":
		h.runPing(ctx)
		h.runDNS(ctx)
		h.runSpeed(ctx)
		h.runInfo(ctx)
	case "help", "--help", "-h":
		h.presenter.PrintHelp()
		return
	default:
		fmt.Printf("\n  %sUnknown command: %s%s\n", ui.Red, cmd, ui.Reset)
		h.presenter.PrintHelp()
		return
	}

	fmt.Printf("\n%s%s  Done.%s\n\n", ui.Bold, ui.Dim, ui.Reset)
}

func (h *Handler) runPing(ctx context.Context) {
	h.presenter.Section("PING — TCP Latency")
	
	hosts := []struct{ Name, Addr string }{
		{"Google DNS    ", "8.8.8.8:53"},
		{"Cloudflare DNS", "1.1.1.1:53"},
		{"OpenDNS       ", "208.67.222.222:53"},
		{"Quad9 DNS     ", "9.9.9.9:53"},
	}

	progress := make(chan string)
	go func() {
		for name := range progress {
			fmt.Printf("  Pinging %s%s%s ", ui.Cyan, name, ui.Reset)
			// Small hack for the dots, ideally use another channel for dots
			for i := 0; i < 5; i++ {
				fmt.Print(".")
				// In a real app we'd need more complex progress tracking
			}
			fmt.Println()
		}
	}()

	results := h.useCase.RunPing(ctx, hosts, 5, progress)
	close(progress)
	h.presenter.PrintPingResults(results)
}

func (h *Handler) runDNS(ctx context.Context) {
	h.presenter.Section("DNS — Resolution Latency")
	domains := []string{
		"google.com", "github.com", "cloudflare.com",
		"youtube.com", "wikipedia.org", "stackoverflow.com",
	}
	results := h.useCase.RunDNS(ctx, domains)
	h.presenter.PrintDNSResults(results)
}

func (h *Handler) runSpeed(ctx context.Context) {
	h.presenter.Section("SPEED — Download (Cloudflare CDN)")
	tests := []struct {
		Label string
		URL   string
	}{
		{"Small  (1 MB) ", "https://speed.cloudflare.com/__down?bytes=1000000"},
		{"Medium (10 MB)", "https://speed.cloudflare.com/__down?bytes=10000000"},
		{"Large  (25 MB)", "https://speed.cloudflare.com/__down?bytes=25000000"},
	}

	fmt.Println("  (Downloads starting...)")
	results := h.useCase.RunSpeed(ctx, tests)
	h.presenter.PrintSpeedResults(results)
}

func (h *Handler) runInfo(ctx context.Context) {
	h.presenter.Section("INFO — Network Details")
	info, _ := h.useCase.RunInfo(ctx)
	h.presenter.PrintNetworkInfo(info)
}
