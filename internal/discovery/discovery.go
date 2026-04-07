package discovery

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/grandcat/zeroconf"
)

type Config struct {
	CIDRs   string
	Ports   string
	Timeout time.Duration
	Iface   string
}

type Asset struct {
	Name        string            `json:"name"`
	ServiceType string            `json:"serviceType"`
	ServiceName string            `json:"serviceName"`
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	TTL         uint32            `json:"ttl"`
	IPv4        []string          `json:"ipv4,omitempty"`
	IPv6        []string          `json:"ipv6,omitempty"`
	TXT         map[string]string `json:"txt,omitempty"`
	TXTOrder    []string          `json:"-"`
	Banner      map[string]string `json:"banner,omitempty"`
}

func Run(ctx context.Context, cfg Config) ([]*Asset, []string, error) {
	portMatcher, err := ParsePorts(cfg.Ports)
	if err != nil {
		return nil, nil, err
	}

	cidrMatcher, err := ParseCIDRs(cfg.CIDRs)
	if err != nil {
		return nil, nil, err
	}

	serviceTypes, err := browseServiceTypes(ctx, cfg.Iface, cfg.Timeout)
	if err != nil {
		return nil, nil, err
	}

	assetsByKey := make(map[string]*Asset)
	for _, serviceType := range serviceTypes {
		entries, err := browseEntries(ctx, cfg.Iface, serviceType, cfg.Timeout)
		if err != nil {
			return nil, nil, err
		}
		for _, entry := range entries {
			asset := toAsset(entry)
			if !portMatcher.Match(asset.Port) {
				continue
			}
			if !cidrMatcher.MatchAny(asset.IPv4, asset.IPv6) {
				continue
			}
			key := assetKey(asset)
			if existing, ok := assetsByKey[key]; ok {
				mergeAsset(existing, asset)
				continue
			}
			assetsByKey[key] = asset
		}
	}

	assets := make([]*Asset, 0, len(assetsByKey))
	for _, asset := range assetsByKey {
		sort.Strings(asset.IPv4)
		sort.Strings(asset.IPv6)
		assets = append(assets, asset)
	}

	sort.Slice(assets, func(i, j int) bool {
		if assets[i].Port != assets[j].Port {
			return assets[i].Port < assets[j].Port
		}
		if assets[i].ServiceName != assets[j].ServiceName {
			return assets[i].ServiceName < assets[j].ServiceName
		}
		return assets[i].Name < assets[j].Name
	})

	return assets, serviceTypes, nil
}

func newResolver(ifaceName string) (*zeroconf.Resolver, error) {
	if strings.TrimSpace(ifaceName) == "" {
		return zeroconf.NewResolver()
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("lookup interface %q: %w", ifaceName, err)
	}

	return zeroconf.NewResolver(zeroconf.SelectIfaces([]net.Interface{*iface}))
}

func browseServiceTypes(ctx context.Context, ifaceName string, timeout time.Duration) ([]string, error) {
	entries, err := browseEntries(ctx, ifaceName, "_services._dns-sd._udp", timeout)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	serviceTypes := make([]string, 0, len(entries))
	for _, entry := range entries {
		serviceType := normalizeServiceType(entry.Instance)
		if serviceType == "" {
			continue
		}
		if _, ok := seen[serviceType]; ok {
			continue
		}
		seen[serviceType] = struct{}{}
		serviceTypes = append(serviceTypes, serviceType)
	}
	sort.Strings(serviceTypes)
	return serviceTypes, nil
}

func browseEntries(ctx context.Context, ifaceName string, service string, timeout time.Duration) ([]*zeroconf.ServiceEntry, error) {
	resolver, err := newResolver(ifaceName)
	if err != nil {
		return nil, err
	}

	browseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ch := make(chan *zeroconf.ServiceEntry)
	if err := resolver.Browse(browseCtx, service, "local.", ch); err != nil {
		return nil, fmt.Errorf("browse %s: %w", service, err)
	}

	var entries []*zeroconf.ServiceEntry
	for entry := range ch {
		entries = append(entries, entry)
	}
	return entries, nil
}

func toAsset(entry *zeroconf.ServiceEntry) *Asset {
	txt, txtOrder := parseTXT(entry.Text)
	asset := &Asset{
		Name:        strings.TrimSpace(decodeEscapedName(entry.Instance)),
		ServiceType: normalizeServiceType(entry.Service),
		ServiceName: friendlyServiceName(entry.Service),
		Host:        strings.TrimSuffix(entry.HostName, "."),
		Port:        entry.Port,
		TTL:         entry.TTL,
		TXT:         txt,
		TXTOrder:    txtOrder,
	}

	for _, ip := range entry.AddrIPv4 {
		asset.IPv4 = append(asset.IPv4, ip.String())
	}
	for _, ip := range entry.AddrIPv6 {
		asset.IPv6 = append(asset.IPv6, ip.String())
	}
	return asset
}

func parseTXT(records []string) (map[string]string, []string) {
	if len(records) == 0 {
		return nil, nil
	}

	values := make(map[string]string, len(records))
	order := make([]string, 0, len(records))
	for _, record := range records {
		if record == "" {
			continue
		}
		parts := strings.SplitN(record, "=", 2)
		key := parts[0]
		value := ""
		if len(parts) == 2 {
			value = parts[1]
		}
		values[key] = value
		order = append(order, key)
	}
	return values, order
}

func assetKey(asset *Asset) string {
	return strings.Join([]string{
		asset.ServiceType,
		strconv.Itoa(asset.Port),
		asset.Name,
		asset.Host,
		strings.Join(asset.IPv4, ","),
		strings.Join(asset.IPv6, ","),
	}, "|")
}

func mergeAsset(dst, src *Asset) {
	dst.IPv4 = uniqueAppend(dst.IPv4, src.IPv4...)
	dst.IPv6 = uniqueAppend(dst.IPv6, src.IPv6...)
	if dst.TTL == 0 {
		dst.TTL = src.TTL
	}
	if len(dst.TXT) == 0 && len(src.TXT) > 0 {
		dst.TXT = src.TXT
		dst.TXTOrder = src.TXTOrder
	}
}

func uniqueAppend(base []string, values ...string) []string {
	seen := make(map[string]struct{}, len(base))
	for _, value := range base {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		base = append(base, value)
		seen[value] = struct{}{}
	}
	return base
}

func normalizeServiceType(service string) string {
	service = strings.TrimSuffix(service, ".")
	service = strings.TrimSuffix(service, ".local")
	return service
}

func friendlyServiceName(service string) string {
	service = normalizeServiceType(service)
	service = strings.TrimPrefix(service, "_")
	if idx := strings.Index(service, "._"); idx >= 0 {
		service = service[:idx]
	}
	return service
}

func decodeEscapedName(value string) string {
	if !strings.Contains(value, `\`) {
		return value
	}

	buf := make([]byte, 0, len(value))
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i == len(value)-1 {
			buf = append(buf, value[i])
			continue
		}

		if i+3 < len(value) && isDigit(value[i+1]) && isDigit(value[i+2]) && isDigit(value[i+3]) {
			n, err := strconv.Atoi(value[i+1 : i+4])
			if err == nil && n >= 0 && n <= 255 {
				buf = append(buf, byte(n))
				i += 3
				continue
			}
		}

		buf = append(buf, value[i+1])
		i++
	}

	if !utf8.Valid(buf) {
		return string(buf)
	}
	return string(buf)
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
