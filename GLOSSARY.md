# CLIProxyAPI

A proxy gateway that aggregates OAuth credentials and API keys across upstream AI providers into unified OpenAI, Gemini, Claude, and Codex compatible endpoints.

## Language

### Model Identity and Catalog

**Upstream Model Name**:
The canonical model identifier recognized by an upstream provider endpoint when issuing inference requests (configured via `name`).
_Avoid_: Target model, raw model name, backend model

**Client Model ID**:
The model identifier exposed to downstream clients for routing and selection (configured via `alias`, or defaulting to the Upstream Model Name when omitted, optionally namespaced by a Credential Prefix).
_Avoid_: Alias name, route name, public model name

**Display Name**:
The human-readable label presented in model catalog responses for UI display without affecting routing identity (configured via `display-name`).
_Avoid_: Title, friendly name, UI alias

**Credential Prefix**:
An optional namespace prepended to every Client Model ID registered by a specific credential so downstream clients can target or group that credential explicitly (configured via `prefix`).
_Avoid_: Group prefix, provider namespace

### Model Metadata Sources

**Static Model Catalog**:
The built-in baseline metadata definitions bundled with the gateway for known provider models, including context window, token limits, modalities, and reasoning capabilities.
_Avoid_: Hardcoded models, local model table

**Upstream Endpoint Metadata**:
Live model specifications retrieved from a configured upstream provider's model listing endpoint, used to enrich or override baseline catalog attributes and discover custom upstream models.
_Avoid_: Remote model info, probed model specs

**Explicit Model Whitelist**:
The list of models explicitly declared under a credential's `models` configuration, which restricts that credential to serving only the declared models unless excluded individually.
_Avoid_: Custom model list, configured model override

### Antigravity Upstream Evasion and Resilience

**Client Wire Fingerprint**:
The combination of HTTP protocol version, TLS ALPN negotiation state, and `User-Agent` metadata tuple (version, platform, and Piper changelist) that identifies an outbound request as originating from the official Antigravity client binary.
_Avoid_: Spoofed UA, browser fingerprint, client header profile

**System Instruction Cloaking**:
The selective transformation of third-party agent harness declarations and brand identifiers inside outbound system instructions—via zero-width character insertion or canonical identity substitution—without mutating user conversation messages or tool schemas.
_Avoid_: Prompt sanitization, global word filtering, payload scrubbing

**Service-Scoped Suspension**:
An upstream `403 PERMISSION_DENIED` rejection (`TOS_VIOLATION` or `VALIDATION_REQUIRED`) that disables Cloud Code inference for a specific OAuth credential until manual user appeal or verification, while leaving the underlying Google account and subscription intact.
_Avoid_: Account ban, quota exhaustion, transient 403

**Failover Contagion**:
The cascading flagging of multiple OAuth credentials in a pool caused by replaying a rejected request across sibling accounts in rapid succession without backoff jitter.
_Avoid_: Retry storm, credential cascade, pool poisoning

### AIStudio2API Native Plugin Integration

**AIStudio2API Native Plugin**:
The native C ABI v1 / Schema v10 plugin (`jianli9158/cpa-plugin-aistudio2api`) loaded dynamically by CLIProxyAPI to serve pooled Google AI Studio accounts directly without external sidecar processes or custom UI wrappers.
_Avoid_: Magpie sidecar, inner proxy, external daemon

**AIStudio2API Auth Directory**:
The standardized storage root at `/root/.cli-proxy-api` (both on Cloud Run mounted via GCS FUSE and on local desktop via user home / config probing) containing accounts, session cookies, and plugin binaries without requiring symlinks or legacy `/root/.antigravity_tools` paths.
_Avoid_: Magpie data dir, antigravity_tools mount, sidecar runtime

**Account State Bundle**:
A portable snapshot of Google AI Studio account session cookies, credentials, and state transferred into `/root/.cli-proxy-api/aistudio2api/accounts/<email>/` and registered via auth files (`aistudio2api-<email>.json`).
_Avoid_: Cookie dump, sidecar bundle, raw storage state

### Automated Upstream Convergence

**Upstream Sync Cycle**:
A scheduled or manually dispatched reconciliation run that merges upstream `router-for-me/CLIProxyAPI` commits before publishing a refreshed container image and release package.
_Avoid_: Dual-upstream sync, auto-update job, fork sync

**Agentic Conflict Resolution**:
An automated repair loop invoked during an Upstream Sync Cycle when a git merge conflict or build/test failure occurs, using a tool-calling LLM agent bounded by `gofmt`, `go build ./cmd/server`, and `go test ./...` verification gates.
_Avoid_: AI merge, auto-fix script, blind conflict resolution

**Quarantine Sync Branch**:
An isolated branch pushed when an Upstream Sync Cycle fails verification after exhausting Agentic Conflict Resolution, preserving the unmerged state and diagnostic logs without advancing `main` or publishing a container image.
_Avoid_: Failed branch, broken main, dirty sync


