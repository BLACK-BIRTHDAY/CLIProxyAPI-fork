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
