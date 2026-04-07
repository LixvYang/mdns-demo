package output

import (
	"strings"
	"testing"

	"github.com/LixvYang/mdns-demo/internal/discovery"
)

func TestTextUsesAssetProtocol(t *testing.T) {
	report := Report{
		Services: []*discovery.Asset{
			{
				ServiceName: "sleep-proxy",
				Protocol:    "udp",
				Port:        5353,
				Name:        "proxy",
			},
		},
	}

	got := Text(report)
	if !strings.Contains(got, "5353/udp sleep-proxy:\n") {
		t.Fatalf("expected udp header, got:\n%s", got)
	}
}

func TestTextOmitsZeroPortPrefix(t *testing.T) {
	report := Report{
		Services: []*discovery.Asset{
			{
				ServiceName: "device-info",
				Protocol:    "tcp",
				Port:        0,
				Name:        "device",
			},
		},
	}

	got := Text(report)
	if !strings.Contains(got, "device-info:\n") {
		t.Fatalf("expected zero-port header without prefix, got:\n%s", got)
	}
	if strings.Contains(got, "0/tcp device-info:") {
		t.Fatalf("unexpected zero-port prefix, got:\n%s", got)
	}
}
