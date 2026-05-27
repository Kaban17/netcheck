package cli

import (
	"fmt"
	"math"
	"strings"
	"time"

	"netcheck/internal/domain"
	"netcheck/internal/ui"
)

type Presenter struct{}

func NewPresenter() *Presenter {
	return &Presenter{}
}

func (p *Presenter) PrintBanner() {
	fmt.Printf("\n%s%s", ui.Bold, ui.Cyan)
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Println("  ║         netcheck — v1.0.0            ║")
	fmt.Println("  ║   Network diagnostics CLI tool       ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Printf("%s\n", ui.Reset)
}

func (p *Presenter) PrintHelp() {
	fmt.Printf("%s%sUsage:%s netcheck <command>\n\n", ui.Bold, ui.White, ui.Reset)
	cmds := [][2]string{
		{"ping", "TCP latency to popular DNS servers (min/avg/max/jitter)"},
		{"dns", "DNS resolution time for popular domains"},
		{"speed", "Download speed test via Cloudflare"},
		{"info", "Public IP, ISP info, local network interfaces"},
		{"all", "Run all checks (default)"},
		{"help", "Show this help message"},
	}
	for _, c := range cmds {
		fmt.Printf("  %s%-8s%s  %s%s%s\n", ui.Bold+ui.Cyan, c[0], ui.Reset, ui.Dim, c[1], ui.Reset)
	}
	fmt.Println()
}

func (p *Presenter) Section(title string) {
	titleLength := len([]rune(title))
	totalWidth := 60
	rightChars := totalWidth - titleLength - 5
	if rightChars < 3 {
		rightChars = 3
	}
	rightBar := strings.Repeat("─", rightChars)
	fmt.Printf("\n%s┌─ %s %s┐%s\n", ui.Bold+ui.Blue, title, rightBar, ui.Reset)
}

func (p *Presenter) PrintPingResults(results []domain.PingResult) {
	fmt.Println()
	fmt.Printf("  %-18s %8s %8s %8s %8s %7s\n",
		ui.Bold+"HOST"+ui.Reset, ui.Bold+"MIN"+ui.Reset, ui.Bold+"AVG"+ui.Reset, ui.Bold+"MAX"+ui.Reset, ui.Bold+"JITTER"+ui.Reset, ui.Bold+"LOSS"+ui.Reset)
	fmt.Printf("  %s\n", strings.Repeat("─", 60))

	for _, r := range results {
		if len(r.Samples) == 0 {
			fmt.Printf("  %-18s %s%-52s%s\n", r.Name, ui.Red, "unreachable", ui.Reset)
			continue
		}
		mn, avg, mx, jit := p.calculateStats(r.Samples)
		loss := float64(r.Errors) / float64(len(r.Samples)+r.Errors) * 100
		lossStr := fmt.Sprintf("%.0f%%", loss)
		lossCol := ui.Green
		if loss > 0 {
			lossCol = ui.Red
		}
		col := p.rttColor(avg)
		fmt.Printf("  %-18s %s%7.1fms%s %s%7.1fms%s %7.1fms %7.1fms %s%6s%s\n",
			r.Name,
			col, float64(mn)/float64(time.Millisecond), ui.Reset,
			col, float64(avg)/float64(time.Millisecond), ui.Reset,
			float64(mx)/float64(time.Millisecond),
			float64(jit)/float64(time.Millisecond),
			lossCol, lossStr, ui.Reset,
		)
	}
}

func (p *Presenter) PrintDNSResults(results []domain.DNSResult) {
	fmt.Printf("\n  %-22s %8s  %s\n", ui.Bold+"DOMAIN"+ui.Reset, ui.Bold+"TIME"+ui.Reset, ui.Bold+"IPS"+ui.Reset)
	fmt.Printf("  %s\n", strings.Repeat("─", 60))

	for _, r := range results {
		if !r.Success {
			fmt.Printf("  %-22s %s%8s%s  %s\n", r.Host, ui.Red, "FAIL", ui.Reset, "Lookup failed")
			continue
		}

		col := ui.Green
		if r.Elapsed > 200*time.Millisecond {
			col = ui.Yellow
		}
		if r.Elapsed > 500*time.Millisecond {
			col = ui.Red
		}

		shown := r.IPs
		suffix := ""
		if len(shown) > 3 {
			shown = shown[:3]
			suffix = fmt.Sprintf(" +%d more", len(r.IPs)-3)
		}

		fmt.Printf("  %-22s %s%7.1fms%s  %s%s%s\n",
			r.Host, col, float64(r.Elapsed)/float64(time.Millisecond), ui.Reset,
			ui.Dim, strings.Join(shown, ", ")+suffix, ui.Reset,
		)
	}
}

func (p *Presenter) PrintSpeedResults(results []domain.SpeedResult) {
	fmt.Printf("\n  %-18s %10s %10s %12s\n",
		ui.Bold+"TEST"+ui.Reset, ui.Bold+"SIZE"+ui.Reset, ui.Bold+"TIME"+ui.Reset, ui.Bold+"SPEED"+ui.Reset)
	fmt.Printf("  %s\n", strings.Repeat("─", 54))

	var bestMbps float64
	for _, r := range results {
		if r.MbPerSec == 0 {
			fmt.Printf("  %-18s %s%44s%s\n", r.Label, ui.Red, "FAILED", ui.Reset)
			continue
		}

		if r.MbPerSec > bestMbps {
			bestMbps = r.MbPerSec
		}
		col := p.speedColor(r.MbPerSec)
		sizeMB := float64(r.Bytes) / 1_000_000

		fmt.Printf("  %-18s %9.1f MB %9.2fs %s%10.2f Mbps%s\n",
			r.Label, sizeMB, r.Elapsed.Seconds(), col, r.MbPerSec, ui.Reset)
	}

	fmt.Println()
	verdict := ""
	switch {
	case bestMbps >= 100:
		verdict = ui.Green + "🚀 Excellent — great for 4K streaming, gaming, conferencing"
	case bestMbps >= 50:
		verdict = ui.Green + "✅ Very Good — handles most tasks with ease"
	case bestMbps >= 25:
		verdict = ui.Yellow + "⚡ Good — suitable for HD streaming and work"
	case bestMbps >= 5:
		verdict = ui.Yellow + "⚠️  Fair — basic browsing and video calls"
	default:
		verdict = ui.Red + "🐌 Slow — may struggle with streaming"
	}
	fmt.Printf("  %s%s%s\n", ui.Bold, verdict, ui.Reset)
}

func (p *Presenter) PrintNetworkInfo(info domain.NetworkInfo) {
	fmt.Println()
	if info.PublicIP.IP != "" {
		fmt.Printf("  %sPublic IP  :%s %s%s%s\n", ui.Bold, ui.Reset, ui.Cyan, info.PublicIP.IP, ui.Reset)
		fmt.Printf("  %sLocation   :%s %s, %s (%s)\n", ui.Bold, ui.Reset, info.PublicIP.City, info.PublicIP.Region, info.PublicIP.Country)
		fmt.Printf("  %sISP / Org  :%s %s\n", ui.Bold, ui.Reset, info.PublicIP.Org)
	} else {
		fmt.Printf("  %sPublic IP  :%s %scould not fetch%s\n", ui.Bold, ui.Reset, ui.Red, ui.Reset)
	}

	fmt.Println()
	fmt.Printf("  %sLocal Interfaces:%s\n", ui.Bold, ui.Reset)
	for _, iface := range info.Interfaces {
		fmt.Printf("  %s  %-12s%s %s%s%s\n", ui.Cyan+ui.Bold, iface.Name, ui.Reset, ui.Dim, strings.Join(iface.IPs, "  "), ui.Reset)
		if len(iface.Flags) > 0 {
			fmt.Printf("  %s  %-12s%s %s[%s]%s\n", "", "", "", ui.Dim, strings.Join(iface.Flags, ", "), ui.Reset)
		}
	}

	fmt.Println()
	fmt.Printf("  %sDefault Gateway check:%s ", ui.Bold, ui.Reset)
	if info.Gateway != "" {
		if info.GatewayRTT > 0 {
			fmt.Printf("%s%s%s  (%.1fms)\n", ui.Green, info.Gateway, ui.Reset, float64(info.GatewayRTT)/float64(time.Millisecond))
		} else {
			fmt.Printf("%s%s%s\n", ui.Cyan, info.Gateway, ui.Reset)
		}
	} else {
		fmt.Printf("%snot detected%s\n", ui.Yellow, ui.Reset)
	}
}

func (p *Presenter) calculateStats(samples []time.Duration) (min, avg, max, jitter time.Duration) {
	if len(samples) == 0 {
		return
	}
	sum := time.Duration(0)
	min = samples[0]
	max = samples[0]
	for _, s := range samples {
		sum += s
		if s < min {
			min = s
		}
		if s > max {
			max = s
		}
	}
	avg = sum / time.Duration(len(samples))
	var dev float64
	for _, s := range samples {
		d := math.Abs(float64(s - avg))
		dev += d
	}
	jitter = time.Duration(dev / float64(len(samples)))
	return
}

func (p *Presenter) rttColor(d time.Duration) string {
	switch {
	case d < 30*time.Millisecond:
		return ui.Green
	case d < 80*time.Millisecond:
		return ui.Yellow
	default:
		return ui.Red
	}
}

func (p *Presenter) speedColor(mbps float64) string {
	switch {
	case mbps >= 50:
		return ui.Green
	case mbps >= 10:
		return ui.Yellow
	default:
		return ui.Red
	}
}
