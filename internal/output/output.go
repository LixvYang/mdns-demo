package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/LixvYang/mdns-demo/internal/discovery"
)

type Report struct {
	Services []*discovery.Asset `json:"services"`
	Answers  Answers            `json:"answers"`
}

type Answers struct {
	PTR []string `json:"PTR"`
}

func Text(report Report) string {
	var b strings.Builder
	b.WriteString("services:\n")
	for _, asset := range report.Services {
		writeServiceHeader(&b, asset)
		fmt.Fprintf(&b, "Name=%s\n", asset.Name)
		if len(asset.IPv4) > 0 {
			fmt.Fprintf(&b, "IPv4=%s\n", strings.Join(asset.IPv4, ","))
		}
		if len(asset.IPv6) > 0 {
			fmt.Fprintf(&b, "IPv6=%s\n", strings.Join(asset.IPv6, ","))
		}
		if asset.Host != "" {
			fmt.Fprintf(&b, "Hostname=%s\n", asset.Host)
		}
		fmt.Fprintf(&b, "TTL=%d\n", asset.TTL)

		if line := formatPairs(asset.TXTOrder, asset.TXT); line != "" {
			b.WriteString(line)
			b.WriteByte('\n')
		}

		if line := formatPairs(sortedKeys(asset.Banner), asset.Banner); line != "" {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}

	b.WriteString("answers:\n")
	b.WriteString("PTR:\n")
	for _, ptr := range report.Answers.PTR {
		if !strings.HasSuffix(ptr, ".local") {
			ptr += ".local"
		}
		b.WriteString(ptr)
		if !strings.HasSuffix(ptr, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func writeServiceHeader(b *strings.Builder, asset *discovery.Asset) {
	if asset.Port <= 0 {
		fmt.Fprintf(b, "%s:\n", asset.ServiceName)
		return
	}

	protocol := asset.Protocol
	if protocol == "" {
		protocol = "tcp"
	}
	fmt.Fprintf(b, "%d/%s %s:\n", asset.Port, protocol, asset.ServiceName)
}

func formatPairs(keys []string, values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(keys))
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := values[key]; !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, values[key]))
		seen[key] = struct{}{}
	}
	for key, value := range values {
		if _, ok := seen[key]; ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, ",")
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
