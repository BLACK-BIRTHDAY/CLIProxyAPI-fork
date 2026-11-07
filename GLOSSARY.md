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

### MAGPIE Integration

**MAGPIE Backend**:
The managed Google AI Studio upstream service paired with the gateway to execute inference across pooled Google AI Studio accounts.
_Avoid_: Studio plugin, inner proxy, web backend

**MAGPIE Model**:
Any model served by the MAGPIE Backend and exposed on the unified gateway under the `(MAGPIE)/` namespace for both Client Model ID and Display Name so downstream clients group it separately while preserving native Gemini capability detection.
_Avoid_: Studio model, prefixed model

**Account State Bundle**:
A portable snapshot of MAGPIE Google account session states, browser fingerprints, model catalogs, and WAA interpreters transferred from a desktop login environment into a headless cloud deployment.
_Avoid_: Cookie dump, auth archive, raw storage state

**Bidirectional Tool Name Alias**:
A deterministic per-request mapping between a downstream client tool identifier exceeding Google AI Studio's length limit and a collision-free shortened upstream identifier, applied outbound across declarations and conversation history and reversed inbound on responses.
_Avoid_: One-way tool truncation, global tool registry

**Malformed Empty Upstream Reply**:
An upstream completion response that terminates with a malformed-generation finish reason without emitting any visible text or tool calls, which the gateway intercepts before streaming to the client and silently retries.
_Avoid_: Empty stream bug, continuation prompt

