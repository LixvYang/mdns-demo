package discovery

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type PortMatcher struct {
	allowAll bool
	ranges   [][2]int
}

func ParsePorts(spec string) (PortMatcher, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" || spec == "*" {
		return PortMatcher{allowAll: true}, nil
	}

	matcher := PortMatcher{}
	for _, item := range strings.Split(spec, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(item, "-") {
			parts := strings.SplitN(item, "-", 2)
			start, err := parsePort(parts[0])
			if err != nil {
				return matcher, err
			}
			end, err := parsePort(parts[1])
			if err != nil {
				return matcher, err
			}
			if start > end {
				start, end = end, start
			}
			matcher.ranges = append(matcher.ranges, [2]int{start, end})
			continue
		}

		port, err := parsePort(item)
		if err != nil {
			return matcher, err
		}
		matcher.ranges = append(matcher.ranges, [2]int{port, port})
	}

	return matcher, nil
}

func (m PortMatcher) Match(port int) bool {
	if m.allowAll {
		return true
	}
	for _, r := range m.ranges {
		if port >= r[0] && port <= r[1] {
			return true
		}
	}
	return false
}

type CIDRMatcher struct {
	nets []*net.IPNet
}

func ParseCIDRs(spec string) (CIDRMatcher, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return CIDRMatcher{}, nil
	}

	m := CIDRMatcher{}
	for _, item := range strings.Split(spec, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		_, ipnet, err := net.ParseCIDR(item)
		if err != nil {
			return m, fmt.Errorf("parse cidr %q: %w", item, err)
		}
		m.nets = append(m.nets, ipnet)
	}
	return m, nil
}

func (m CIDRMatcher) MatchAny(ipv4, ipv6 []string) bool {
	if len(m.nets) == 0 {
		return true
	}
	for _, raw := range ipv4 {
		if m.match(raw) {
			return true
		}
	}
	for _, raw := range ipv6 {
		if m.match(raw) {
			return true
		}
	}
	return false
}

func (m CIDRMatcher) match(raw string) bool {
	ip := net.ParseIP(raw)
	if ip == nil {
		return false
	}
	for _, ipnet := range m.nets {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

func parsePort(raw string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("parse port %q: %w", raw, err)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port %d out of range", port)
	}
	return port, nil
}
