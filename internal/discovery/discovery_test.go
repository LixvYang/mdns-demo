package discovery

import (
	"testing"
	"time"
)

func TestServiceProtocol(t *testing.T) {
	cases := map[string]string{
		"_http._tcp":        "tcp",
		"_sleep-proxy._udp": "udp",
		"_device-info._tcp": "tcp",
		"_unknown":          "",
	}

	for input, want := range cases {
		if got := serviceProtocol(input); got != want {
			t.Fatalf("serviceProtocol(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestServiceTypeBudgetBounds(t *testing.T) {
	if got := serviceTypeBudget(500 * time.Millisecond); got != 500*time.Millisecond {
		t.Fatalf("expected full budget for short timeout, got %s", got)
	}
	if got := serviceTypeBudget(5 * time.Second); got != 1250*time.Millisecond {
		t.Fatalf("expected quarter budget for 5s timeout, got %s", got)
	}
	if got := serviceTypeBudget(20 * time.Second); got != 2*time.Second {
		t.Fatalf("expected capped budget for long timeout, got %s", got)
	}
}
