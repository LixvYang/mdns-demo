package probe

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/LixvYang/mdns-demo/internal/discovery"
)

var titlePattern = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

func Run(ctx context.Context, asset *discovery.Asset) map[string]string {
	banner := make(map[string]string)

	if line := probeHTTP(ctx, asset, false); len(line) > 0 {
		for k, v := range line {
			banner[k] = v
		}
	}

	if shouldTryHTTPS(asset) {
		if line := probeHTTP(ctx, asset, true); len(line) > 0 {
			for k, v := range line {
				banner[k] = v
			}
		}
	}

	if len(banner) == 0 {
		if line := probeTLS(ctx, asset); len(line) > 0 {
			for k, v := range line {
				banner[k] = v
			}
		}
	}

	if len(banner) == 0 {
		if line := probeRawBanner(ctx, asset); len(line) > 0 {
			for k, v := range line {
				banner[k] = v
			}
		}
	}

	if len(banner) == 0 {
		return nil
	}
	return banner
}

func shouldTryHTTPS(asset *discovery.Asset) bool {
	if strings.Contains(asset.ServiceType, "_https._tcp") {
		return true
	}
	if asset.Port == 443 || asset.Port == 8443 {
		return true
	}
	if strings.EqualFold(asset.TXT["accessType"], "https") {
		return true
	}
	return false
}

func shouldTryHTTP(asset *discovery.Asset) bool {
	if strings.Contains(asset.ServiceType, "_http._tcp") {
		return true
	}
	switch asset.Port {
	case 80, 81, 8080, 8000, 5000:
		return true
	default:
		return false
	}
}

func probeHTTP(ctx context.Context, asset *discovery.Asset, secure bool) map[string]string {
	if secure && !shouldTryHTTPS(asset) {
		return nil
	}
	if !secure && !shouldTryHTTP(asset) {
		return nil
	}

	targetIP := firstTargetIP(asset)
	if targetIP == "" {
		return nil
	}

	scheme := "http"
	if secure {
		scheme = "https"
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(reqCtx context.Context, network, addr string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(reqCtx, "tcp", net.JoinHostPort(targetIP, portString(asset.Port)))
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	host := targetIP
	if asset.Host != "" {
		host = asset.Host
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, scheme+"://"+host+"/", nil)
	if err != nil {
		return nil
	}
	if asset.Host != "" {
		req.Host = asset.Host
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	banner := map[string]string{
		"httpScheme": scheme,
		"httpStatus": resp.Status,
		"path":       resp.Request.URL.Path,
	}
	if server := strings.TrimSpace(resp.Header.Get("Server")); server != "" {
		banner["httpServer"] = server
	}
	if location := strings.TrimSpace(resp.Header.Get("Location")); location != "" {
		banner["httpLocation"] = location
	}
	if matches := titlePattern.FindSubmatch(body); len(matches) == 2 {
		title := strings.TrimSpace(htmlStrip(string(matches[1])))
		if title != "" {
			banner["httpTitle"] = title
		}
	}
	return banner
}

func probeTLS(ctx context.Context, asset *discovery.Asset) map[string]string {
	targetIP := firstTargetIP(asset)
	if targetIP == "" {
		return nil
	}

	var d tls.Dialer
	d.Config = &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         asset.Host,
	}

	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(targetIP, portString(asset.Port)))
	if err != nil {
		return nil
	}
	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		return nil
	}
	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil
	}

	cert := state.PeerCertificates[0]
	banner := map[string]string{
		"tlsVersion": tlsVersionName(state.Version),
	}
	if cert.Subject.CommonName != "" {
		banner["tlsCN"] = cert.Subject.CommonName
	}
	if len(cert.DNSNames) > 0 {
		banner["tlsDNS"] = strings.Join(cert.DNSNames, "|")
	}
	return banner
}

func probeRawBanner(ctx context.Context, asset *discovery.Asset) map[string]string {
	targetIP := firstTargetIP(asset)
	if targetIP == "" {
		return nil
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(targetIP, portString(asset.Port)))
	if err != nil {
		return nil
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return nil
	}
	raw := sanitizeBanner(string(buf[:n]))
	if raw == "" {
		return nil
	}
	return map[string]string{"rawBanner": raw}
}

func firstTargetIP(asset *discovery.Asset) string {
	if len(asset.IPv4) > 0 {
		return asset.IPv4[0]
	}
	if len(asset.IPv6) > 0 {
		return asset.IPv6[0]
	}
	return ""
}

func portString(port int) string {
	return strconv.Itoa(port)
}

func sanitizeBanner(raw string) string {
	raw = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if r < 32 || r > 126 {
			return -1
		}
		return r
	}, raw)
	raw = strings.TrimSpace(raw)
	if len(raw) > 120 {
		raw = raw[:120]
	}
	return raw
}

func htmlStrip(raw string) string {
	replacer := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ")
	return replacer.Replace(strings.TrimSpace(raw))
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS1.0"
	case tls.VersionTLS11:
		return "TLS1.1"
	case tls.VersionTLS12:
		return "TLS1.2"
	case tls.VersionTLS13:
		return "TLS1.3"
	default:
		return "unknown"
	}
}
