# Security Vulnerabilities — Accepted Findings + Operator Mitigations

This document records security findings (govulncheck, etc.) that have been
manually reviewed and **accepted with documented rationale** rather than
patched. Each entry names the CVE, what makes it not-applicable to our use
case, and any operator-side mitigations required to keep that not-applicable
status valid.

Audit history:
- Mantis #323 — 9 ollama CVEs reviewed and documented (2026-04-25)

---

## github.com/ollama/ollama (indirect via go-rag)

**Status as of 2026-04-25:** all 9 CVEs filed in Mantis #323 are **UNFIXED
upstream** per [pkg.go.dev/vuln](https://pkg.go.dev/vuln/). Pin-bumping does
not resolve any of them. We are on `v0.18.1` indirect; ollama upstream is at
`v0.21.2` (2026-04-23).

**Our usage scope:** the entire workspace imports `github.com/ollama/ollama/api`
from exactly ONE file (`go-rag/ollama.go`). The surface in use is **3 symbols
only**:
- `api.NewClient(baseURL, *http.Client)` — constructor
- `api.Client` — struct value (held as a field by `OllamaClient`)
- `api.EmbedRequest` — embedding-request DTO

**We are a CLIENT** of someone else's Ollama server. We do NOT host an Ollama
server. Most CVEs in the list are server-side code paths that govulncheck's
reachability graph flags because the package is imported, but our actual call
sites do not traverse those paths.

### CVE-by-CVE reachability assessment

| CVE | Description | Reachable from our call graph? | Action |
|---|---|---|---|
| GO-2025-3548 (CVE-2024-12886) | DoS via crafted GZIP | NO — server-side parser | Accept |
| GO-2025-3557 (CVE-2025-0315) | Resource alloc without limits | NO — server-side dispatcher | Accept |
| GO-2025-3558 | Out-of-bounds read | NO — server-side inference | Accept |
| GO-2025-3559 | Divide by zero | NO — server-side inference | Accept |
| GO-2025-3582 | Null pointer deref DoS | NO — server-side handler | Accept |
| GO-2025-3689 | Divide by zero | NO — server-side inference | Accept |
| GO-2025-3695 | Server DoS | NO — server-side handler | Accept |
| GO-2025-3824 (CVE-2025-51471) | Cross-domain token exposure | **CONDITIONAL** — see below | Watch |
| GO-2025-4251 (CVE-2025-63389) | Missing auth on model-mgmt | **OPERATOR-SIDE** — see below | Runbook |

### GO-2025-3824 — token-exposure watch flag

This CVE concerns auth tokens leaking across domain boundaries when Ollama
clients pass authentication. Currently `NewOllamaClient(cfg)` constructs over
plain HTTP/HTTPS without auth headers — the embedding client connects to a
trusted local Ollama instance per the deployment runbook below.

**If we ever add auth-token plumbing to the Ollama client** (e.g. for hosted
Ollama services), re-evaluate this CVE. The reachability flips from NO to YES
the moment we set an Authorization header on `api.NewClient`.

### GO-2025-4251 — operator-side mitigation required

This CVE is a missing authentication / authorization gap on Ollama's
model-management endpoints. The vulnerability is in the **Ollama server**,
not our client code. Our client doesn't expose model-management calls;
operators do via running an Ollama server.

**Operator mitigation (REQUIRED):** see "Ollama deployment" section below.
Operators MUST front their Ollama instance with network-level access controls
or an authentication proxy. This is also Ollama upstream's own recommendation
in the advisory.

### Watch flag

If any of the 9 CVEs gets a fixed version released, re-evaluate:
- Bump `go-rag/go.mod` require for `github.com/ollama/ollama` to the fixed version
- Re-run govulncheck and prune entries from this document accordingly

---

## Ollama deployment — operator runbook

The Ollama instance the agent connects to runs OUTSIDE of our application
boundary. Operators are responsible for these mitigations:

### 1. Network-level isolation (mandatory)

Bind the Ollama server to a private interface or front it with a reverse proxy:

```bash
# OPTION A — localhost-only binding (single-host deployments)
OLLAMA_HOST=127.0.0.1:11434 ollama serve

# OPTION B — private network only (multi-host fleet)
# Bind to the wireguard / tailscale / private-VLAN interface, not 0.0.0.0
OLLAMA_HOST=10.42.0.5:11434 ollama serve
```

**Never** expose Ollama directly to the public internet. GO-2025-4251 makes
model-management operations possible without auth.

### 2. Reverse proxy with auth (recommended for shared deployments)

If multiple agents share an Ollama server, front it with nginx/caddy/traefik
adding HTTP Basic Auth or an authentication proxy (oauth2-proxy, authentik):

```nginx
location /ollama/ {
    auth_basic "Ollama API";
    auth_basic_user_file /etc/nginx/ollama.htpasswd;
    proxy_pass http://10.42.0.5:11434/;
}
```

Configure the agent's `OllamaConfig.Endpoint` to point at the reverse proxy
URL, and add an `Authorization` header to the http.Client passed to
`api.NewClient`. (When that change lands, re-evaluate GO-2025-3824 per
the watch-flag note above.)

### 3. CI-side govulncheck filter

Until upstream Ollama ships fixes for any of the 9 CVEs, CI should suppress
just these specific findings (not blanket-suppress all govulncheck output):

```bash
govulncheck ./... 2>&1 | grep -vE 'GO-2025-(3548|3557|3558|3559|3582|3689|3695|3824|4251)\b'
```

When a CVE gets a fix and we bump past it, drop that CVE ID from the grep
filter so future regressions surface cleanly.

---

## How to add to this document

When a new accepted finding lands:

1. Open a new H2 section named for the dependency
2. Document the reachability + rationale per CVE in a table
3. Add operator-side mitigations if any
4. Update the audit-history bullet at the top with a Mantis ticket reference

**Do NOT add findings here without a Mantis ticket.** Every accepted finding
must have a tracker entry so the rationale is auditable + reviewable.
