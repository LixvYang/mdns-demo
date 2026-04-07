package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/LixvYang/mdns-demo/internal/discovery"
	"github.com/LixvYang/mdns-demo/internal/output"
	"github.com/LixvYang/mdns-demo/internal/probe"
)

func main() {
	cfg, err := parseFlags()
	if err != nil {
		fail(err)
	}

	ctx := context.Background()
	assets, serviceTypes, err := discovery.Run(ctx, discovery.Config{
		CIDRs:   cfg.cidr,
		Ports:   cfg.ports,
		Timeout: cfg.timeout,
		Iface:   cfg.iface,
	})
	if err != nil {
		fail(err)
	}

	probeResults(ctx, assets, cfg.concurrency, cfg.probeTimeout)

	report := output.Report{
		Services: assets,
		Answers: output.Answers{
			PTR: serviceTypes,
		},
	}

	if cfg.json {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fail(err)
		}
		return
	}

	fmt.Print(output.Text(report))
}

type cliConfig struct {
	cidr         string
	ports        string
	timeout      time.Duration
	probeTimeout time.Duration
	concurrency  int
	iface        string
	json         bool
}

func parseFlags() (cliConfig, error) {
	var cfg cliConfig
	flag.StringVar(&cfg.cidr, "cidr", "", "CIDR filter, comma-separated, e.g. 192.168.1.0/24")
	flag.StringVar(&cfg.ports, "ports", "1-65535", "Port filter, e.g. 80,443,5000-6000")
	flag.DurationVar(&cfg.timeout, "timeout", 5*time.Second, "Discovery timeout per browse stage")
	flag.DurationVar(&cfg.probeTimeout, "probe-timeout", 2*time.Second, "Probe timeout per asset")
	flag.IntVar(&cfg.concurrency, "concurrency", 16, "Max concurrent banner probes")
	flag.StringVar(&cfg.iface, "iface", "", "Optional interface name for mDNS browsing")
	flag.BoolVar(&cfg.json, "json", false, "Render JSON output")
	flag.Parse()

	if cfg.concurrency <= 0 {
		return cfg, fmt.Errorf("concurrency must be > 0")
	}
	if strings.TrimSpace(cfg.ports) == "" {
		return cfg, fmt.Errorf("ports cannot be empty")
	}
	if cfg.timeout <= 0 || cfg.probeTimeout <= 0 {
		return cfg, fmt.Errorf("timeouts must be > 0")
	}

	return cfg, nil
}

func probeResults(ctx context.Context, assets []*discovery.Asset, concurrency int, timeout time.Duration) {
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, asset := range assets {
		asset := asset
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() {
				<-sem
				wg.Done()
			}()

			pctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			asset.Banner = probe.Run(pctx, asset)
		}()
	}

	wg.Wait()
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
