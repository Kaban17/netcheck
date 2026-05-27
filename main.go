package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// ─── ANSI ──────────────────────────────────────────────────────────────────
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
	white  = "\033[97m"
)

// ─── DATA TYPES ────────────────────────────────────────────────────────────
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
	Label     string
	Bytes     int64
	Elapsed   time.Duration
	MbPerSec  float64
}

type IPInfo struct {
	IP      string `json:"ip"`
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
	Org     string `json:"org"`
}

// ─── BANNER ────────────────────────────────────────────────────────────────
func printBanner() {
	fmt.Printf("\n%s%s", bold, cyan)
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Println("  ║         netcheck — v1.0.0            ║")
	fmt.Println("  ║   Network diagnostics CLI tool       ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Printf("%s\n", reset)
}

func printHelp() {
	fmt.Printf("%s%sUsage:%s netcheck <command>\n\n", bold, white, reset)
	cmds := [][2]string{
		{"ping", "TCP latency to popular DNS servers (min/avg/max/jitter)"},
		{"dns", "DNS resolution time for popular domains"},
		{"speed", "Download speed test via Cloudflare"},
		{"info", "Public IP, ISP info, local network interfaces"},
		{"all", "Run all checks (default)"},
		{"help", "Show this help message"},
	}
	for _, c := range cmds {
		fmt.Printf("  %s%-8s%s  %s%s%s\n", bold+cyan, c[0], reset, dim, c[1], reset)
	}
	fmt.Println()
}

// ─── SECTION HEADER ────────────────────────────────────────────────────────
func section(title string) {
	// Переводим заголовок в срез рун, чтобы узнать точное КОЛИЧЕСТВО СИМВОЛОВ, а не байт
	titleLength := len([]rune(title))

	// Желаемая общая ширина рамки в символах (например, 60)
	totalWidth := 60

	// Считаем, сколько символов '─' нужно нарисовать справа.
	// Отнимаем длину заголовка и 5 служебных символов (уголок '┌', левая палочка '─', два пробела и правый уголок '┐')
	rightChars := totalWidth - titleLength - 5
	if rightChars < 3 {
		rightChars = 3 // Защита, если заголовок слишком длинный
	}

	// Генерируем правую часть рамки нужной длины
	rightBar := strings.Repeat("─", rightChars)

	// Выводим готовую строку без опасных срезов
	fmt.Printf("\n%s┌─ %s %s┐%s\n", bold+blue, title, rightBar, reset)
}// ─── PING ──────────────────────────────────────────────────────────────────
var pingHosts = []struct{ name, addr string }{
	{"Google DNS    ", "8.8.8.8:53"},
	{"Cloudflare DNS", "1.1.1.1:53"},
	{"OpenDNS       ", "208.67.222.222:53"},
	{"Quad9 DNS     ", "9.9.9.9:53"},
}

func tcpPing(addr string) (time.Duration, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return 0, err
	}
	conn.Close()
	return time.Since(start), nil
}

func stats(samples []time.Duration) (min, avg, max, jitter time.Duration) {
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

	// Jitter = mean deviation from avg
	var dev float64
	for _, s := range samples {
		d := math.Abs(float64(s - avg))
		dev += d
	}
	jitter = time.Duration(dev / float64(len(samples)))
	return
}

func rttColor(d time.Duration) string {
	switch {
	case d < 30*time.Millisecond:
		return green
	case d < 80*time.Millisecond:
		return yellow
	default:
		return red
	}
}

func runPing() {
	section("PING — TCP Latency")
	const samples = 5

	results := make([]PingResult, len(pingHosts))
	for i, h := range pingHosts {
		r := PingResult{Name: h.name, Host: h.addr}
		fmt.Printf("  Pinging %s%s%s ", cyan, h.name, reset)
		for s := 0; s < samples; s++ {
			rtt, err := tcpPing(h.addr)
			if err != nil {
				r.Errors++
			} else {
				r.Samples = append(r.Samples, rtt)
			}
			fmt.Print(".")
			time.Sleep(150 * time.Millisecond)
		}
		fmt.Println()
		results[i] = r
	}

	fmt.Println()
	fmt.Printf("  %-18s %8s %8s %8s %8s %7s\n",
		bold+"HOST"+reset, bold+"MIN"+reset, bold+"AVG"+reset, bold+"MAX"+reset, bold+"JITTER"+reset, bold+"LOSS"+reset)
	fmt.Printf("  %s\n", strings.Repeat("─", 60))

	for _, r := range results {
		if len(r.Samples) == 0 {
			fmt.Printf("  %-18s %s%-52s%s\n", r.Name, red, "unreachable", reset)
			continue
		}
		mn, avg, mx, jit := stats(r.Samples)
		loss := float64(r.Errors) / float64(samples) * 100
		lossStr := fmt.Sprintf("%.0f%%", loss)
		lossCol := green
		if loss > 0 {
			lossCol = red
		}
		col := rttColor(avg)
		fmt.Printf("  %-18s %s%7.1fms%s %s%7.1fms%s %7.1fms %7.1fms %s%6s%s\n",
			r.Name,
			col, float64(mn)/float64(time.Millisecond), reset,
			col, float64(avg)/float64(time.Millisecond), reset,
			float64(mx)/float64(time.Millisecond),
			float64(jit)/float64(time.Millisecond),
			lossCol, lossStr, reset,
		)
	}
}

// ─── DNS ───────────────────────────────────────────────────────────────────
var dnsHosts = []string{
	"google.com", "github.com", "cloudflare.com",
	"youtube.com", "wikipedia.org", "stackoverflow.com",
}

func runDNS() {
	section("DNS — Resolution Latency")
	fmt.Printf("\n  %-22s %8s  %s\n", bold+"DOMAIN"+reset, bold+"TIME"+reset, bold+"IPS"+reset)
	fmt.Printf("  %s\n", strings.Repeat("─", 60))

	for _, host := range dnsHosts {
		start := time.Now()
		addrs, err := net.LookupHost(host)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("  %-22s %s%8s%s  %s\n", host, red, "FAIL", reset, err.Error())
			continue
		}

		col := green
		if elapsed > 200*time.Millisecond {
			col = yellow
		}
		if elapsed > 500*time.Millisecond {
			col = red
		}

		// Limit displayed IPs
		shown := addrs
		suffix := ""
		if len(shown) > 3 {
			shown = shown[:3]
			suffix = fmt.Sprintf(" +%d more", len(addrs)-3)
		}

		fmt.Printf("  %-22s %s%7.1fms%s  %s%s%s\n",
			host, col, float64(elapsed)/float64(time.Millisecond), reset,
			dim, strings.Join(shown, ", ")+suffix, reset,
		)
	}
}

// ─── SPEED ─────────────────────────────────────────────────────────────────
var speedTests = []struct {
	label string
	url   string
	bytes int64
}{
	{"Small  (1 MB) ", "https://speed.cloudflare.com/__down?bytes=1000000", 1_000_000},
	{"Medium (10 MB)", "https://speed.cloudflare.com/__down?bytes=10000000", 10_000_000},
	{"Large  (25 MB)", "https://speed.cloudflare.com/__down?bytes=25000000", 25_000_000},
}

func measureDownload(url string) (int64, time.Duration, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "netcheck/1.0")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	n, err := io.Copy(io.Discard, resp.Body)
	elapsed := time.Since(start)
	return n, elapsed, err
}

func speedColor(mbps float64) string {
	switch {
	case mbps >= 50:
		return green
	case mbps >= 10:
		return yellow
	default:
		return red
	}
}

func runSpeed() {
	section("SPEED — Download (Cloudflare CDN)")
	fmt.Printf("\n  %-18s %10s %10s %12s\n",
		bold+"TEST"+reset, bold+"SIZE"+reset, bold+"TIME"+reset, bold+"SPEED"+reset)
	fmt.Printf("  %s\n", strings.Repeat("─", 54))

	var bestMbps float64
	for _, t := range speedTests {
		fmt.Printf("  Downloading %-18s ...", t.label)

		n, elapsed, err := measureDownload(t.url)
		if err != nil {
			fmt.Printf("\r  %-18s %s%44s%s\n", t.label, red, "FAILED: "+err.Error(), reset)
			continue
		}

		mbps := float64(n) * 8 / elapsed.Seconds() / 1_000_000
		if mbps > bestMbps {
			bestMbps = mbps
		}
		col := speedColor(mbps)
		sizeMB := float64(n) / 1_000_000

		fmt.Printf("\r  %-18s %9.1f MB %9.2fs %s%10.2f Mbps%s\n",
			t.label, sizeMB, elapsed.Seconds(), col, mbps, reset)
	}

	fmt.Println()
	verdict := ""
	switch {
	case bestMbps >= 100:
		verdict = green + "🚀 Excellent — great for 4K streaming, gaming, conferencing"
	case bestMbps >= 50:
		verdict = green + "✅ Very Good — handles most tasks with ease"
	case bestMbps >= 25:
		verdict = yellow + "⚡ Good — suitable for HD streaming and work"
	case bestMbps >= 5:
		verdict = yellow + "⚠️  Fair — basic browsing and video calls"
	default:
		verdict = red + "🐌 Slow — may struggle with streaming"
	}
	fmt.Printf("  %s%s%s\n", bold, verdict, reset)
}

// ─── INFO ──────────────────────────────────────────────────────────────────
func runInfo() {
	section("INFO — Network Details")
	fmt.Println()

	// Public IP & ISP
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://ipinfo.io/json")
	if err == nil {
		defer resp.Body.Close()
		var info IPInfo
		if json.NewDecoder(resp.Body).Decode(&info) == nil {
			fmt.Printf("  %sPublic IP  :%s %s%s%s\n", bold, reset, cyan, info.IP, reset)
			fmt.Printf("  %sLocation   :%s %s, %s (%s)\n", bold, reset, info.City, info.Region, info.Country)
			fmt.Printf("  %sISP / Org  :%s %s\n", bold, reset, info.Org)
		}
	} else {
		fmt.Printf("  %sPublic IP  :%s %scould not fetch%s\n", bold, reset, red, reset)
	}

	fmt.Println()

	// Local interfaces
	fmt.Printf("  %sLocal Interfaces:%s\n", bold, reset)
	ifaces, _ := net.Interfaces()
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
		fmt.Printf("  %s  %-12s%s %s%s%s\n", cyan+bold, iface.Name, reset, dim, strings.Join(ips, "  "), reset)
		if len(flags) > 0 {
			fmt.Printf("  %s  %-12s%s %s[%s]%s\n", "", "", "", dim, strings.Join(flags, ", "), reset)
		}
	}

	// Traceroute-lite: measure hops to 8.8.8.8 via TTL
	fmt.Println()
	fmt.Printf("  %sDefault Gateway check:%s ", bold, reset)
	gw := detectGateway()
	if gw != "" {
		rtt, err := tcpPing(gw + ":80")
		if err != nil {
			// try port 443
			rtt, err = tcpPing(gw + ":443")
		}
		if err == nil {
			fmt.Printf("%s%s%s  (%.1fms)\n", green, gw, reset, float64(rtt)/float64(time.Millisecond))
		} else {
			fmt.Printf("%s%s%s\n", cyan, gw, reset)
		}
	} else {
		fmt.Printf("%snot detected%s\n", yellow, reset)
	}
}

// detectGateway tries to find the default gateway by connecting to 8.8.8.8:53
// and reading the local address's subnet.
func detectGateway() string {
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
	// Typical assumption: gateway is .1 in the same /24
	gw := fmt.Sprintf("%d.%d.%d.1", ip[0], ip[1], ip[2])
	return gw
}

// ─── MAIN ──────────────────────────────────────────────────────────────────
func main() {
	cmd := "all"
	if len(os.Args) >= 2 {
		cmd = strings.ToLower(os.Args[1])
	}


	printBanner()

	switch cmd {
	case "ping":
		runPing()
	case "dns":
		runDNS()
	case "speed":
		runSpeed()
	case "info":
		runInfo()
	case "all":
		runPing()
		runDNS()
		runSpeed()
		runInfo()
	case "help", "--help", "-h":
		printHelp()
		return
	default:
		fmt.Printf("\n  %sUnknown command: %s%s\n", red, cmd, reset)
		printHelp()
		return
	}

	fmt.Printf("\n%s%s  Done.%s\n\n", bold, dim, reset)
}
