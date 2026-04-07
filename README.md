# mdns-demo

A first-pass Go CLI for discovering mDNS / DNS-SD assets on the local link, filtering them by CIDR and port range, and printing service metadata plus basic banner enrichment.

## What It Does

- Browses `_services._dns-sd._udp.local.`
- Enumerates concrete service types such as `_http._tcp` or `_qdiscover._tcp`
- Collects `PTR`, `SRV`, `TXT`, `A`, and `AAAA`-derived data through `zeroconf`
- Filters results by `--cidr` and `--ports`
- Enriches matched assets with lightweight probes:
  - HTTP / HTTPS metadata
  - TLS certificate details
  - Raw TCP banner when available

## Important Limitation

mDNS is a link-local discovery protocol. This tool discovers services visible from the current network interface and then filters them by CIDR and port range. It is not a general cross-subnet scanner.

## Build

```bash
go build ./...
```

## Usage

```bash
go run . --timeout 2s --probe-timeout 500ms
go run . --cidr 192.168.1.0/24 --ports 80,443,5000-6000
go run . --iface en0 --json
```

Flags:

- `--cidr`: comma-separated CIDR filters
- `--ports`: comma-separated ports or ranges
- `--timeout`: discovery timeout per browse stage
- `--probe-timeout`: active probe timeout per asset
- `--concurrency`: maximum concurrent probes
- `--iface`: optional interface name
- `--json`: render JSON instead of text

## Text Output Shape

```text
services:
5000/tcp http:
Name=example
IPv4=192.168.1.10
Hostname=example.local
TTL=120
path=/
httpStatus=200 OK,httpTitle=Example,httpServer=nginx/1.25.0
answers:
PTR:
_http._tcp.local
```

## Validation

Verified locally on `darwin/arm64` with:

```bash
go build ./...
go run . --timeout 2s --probe-timeout 500ms
```

Observed sample result from the current LAN:

```text
services:
7000/tcp airplay:
Name=lixin’s MacBook Pro
IPv4=192.168.207.0
IPv6=fe80::f84d:89ff:fed5:9468
Hostname=lixins-MacBook-Pro.local
TTL=4500
act=2,acl=0,deviceid=F8:4D:89:5D:AD:4E,...,model=MacBookPro18,3,...
7000/tcp raop:
Name=F84D895DAD4E@lixin’s MacBook Pro
IPv4=192.168.207.0
IPv6=fe80::f84d:89ff:fed5:9468
Hostname=lixins-MacBook-Pro.local
TTL=4500
cn=0,1,2,3,...,am=MacBookPro18,3,...
57179/tcp companion-link:
Name=lixin’s MacBook Pro
IPv4=192.168.207.0
IPv6=fe80::f84d:89ff:fed5:9468
Hostname=lixins-MacBook-Pro.local
TTL=4500
rpMac=0,rpHN=92577cc272b2,rpFl=0x20000,...
answers:
PTR:
_airplay._tcp.local
_companion-link._tcp.local
_raop._tcp.local
```
