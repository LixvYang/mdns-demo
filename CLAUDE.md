# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Goal

Go CLI tool that discovers mDNS/DNS-SD assets on the local network, filters by user-supplied CIDR and port range, and outputs deep banner information.

## Build & Run

```bash
go build -o mdns-demo .
./mdns-demo --cidr 192.168.1.0/24 --ports 1-9000
```

## Architecture

**Implementation approach**: multicast mDNS browsing via `grandcat/zeroconf`, with CIDR and port range applied as post-discovery filters (not per-IP active probing). This is a deliberate constraint — mDNS is link-local only.

**Key dependencies**:
- `github.com/grandcat/zeroconf` — mDNS/DNS-SD browse, handles PTR/SRV/TXT/A/AAAA aggregation

**Current packages**:
- `main.go` — CLI flags and orchestration
- `internal/discovery` — mDNS browsing, CIDR filtering, port filtering
- `internal/probe` — HTTP/TLS/raw banner enrichment
- `internal/output` — text and JSON output

**Banner depth (three layers, in priority order)**:
1. TXT record KV pairs from mDNS response (zero extra probing)
2. Active HTTP `GET /` for `_http._tcp` / `_https._tcp` services — extract Server header, title, status, redirect path
3. TLS certificate metadata or raw TCP banner as fallback

SMB negotiate and AFP handshake are explicitly out of scope.

## Output Format

```
services:
7000/tcp airplay:
Name=...
IPv4=...
IPv6=...
Hostname=...
TTL=...
{txt_key}={txt_val},...
answers:
PTR:
_xxx._tcp.local
```

## Constraints from Task

- No manual edits allowed — all changes via AI tooling only
- Must push to a public GitHub repo when complete
- `gh` CLI is available and authenticated as `LixvYang`
