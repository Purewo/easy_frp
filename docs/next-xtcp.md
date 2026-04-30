# Next Milestone: XTCP

Status: superseded by the implemented room-mode contract in [client-server-rooms.md](client-server-rooms.md) and the [v0.2.0 release notes](releases/v0.2.0.md).

Current direction:

- Do not add local `/v1/xtcp` APIs or `ctl xtcp` commands for room mode.
- Keep the existing local backend focused on direct/public exposure.
- Implement XTCP rooms through `frp-panel server` plus foreground `frp-panel client host` and `frp-panel client join` commands.
- During the first rollout, run a separate room-only frps, for example on port `17000`, and keep the existing production frps for direct/public exposure. Merge later only after room authorization and operations are stable.

The sections below are retained as historical context only; do not implement them as the active plan.

This document is the handoff for the next development round. Do not start by redesigning the current direct-port flow; treat `v0.1.0` as the stable baseline and add XTCP as the next backend capability.

## Goal

Add a user-friendly backend workflow for frp XTCP so two clients can connect through frps-assisted NAT traversal without exposing a public TCP port.

The product goal is:

- one user creates an XTCP service on a local machine,
- another user creates an XTCP visitor that connects to it,
- the UI can present this as a private peer-to-peer access rule,
- all generated frpc config remains inspectable, reloadable, and diagnosable.

## Baseline To Preserve

Do not break the current direct exposure system:

- `/v1/nodes`
- `/v1/ports`
- `/v1/frpc/status`
- `/v1/frpc/reload`
- `/v1/logs`
- CLI `ctl nodes`
- CLI `ctl ports`
- CLI `ctl expose`

The current multi-node architecture is the foundation: enabled rules are grouped by `nodeId`, and each active node owns its own frpc config, process, admin port, and log file.

## frp Concepts To Model

XTCP uses two sides:

- `xtcp` service proxy: runs beside the local service that should be reached.
- `xtcp` visitor proxy: runs beside the client that wants to connect.

The backend should model those as explicit records instead of hiding them in raw config text.

Expected frpc fields to study and support according to frp's official XTCP documentation:

- service `type = "xtcp"`,
- service `secretKey`,
- service `localIP`,
- service `localPort`,
- visitor `type = "xtcp"`,
- visitor `serverName`,
- visitor `secretKey`,
- visitor `bindAddr`,
- visitor `bindPort`.

Confirm current frp field names against the installed frp version before implementation.

## Proposed Backend Model

Add an XTCP rule family separate from `/v1/ports`.

Suggested storage records:

```json
{
  "id": "xtcp_...",
  "nodeId": "default",
  "name": "private-api",
  "role": "server",
  "serverName": "private-api",
  "localIP": "127.0.0.1",
  "localPort": 8080,
  "secretKeySet": true,
  "enabled": true,
  "createdAt": "2026-04-30T00:00:00Z",
  "updatedAt": "2026-04-30T00:00:00Z"
}
```

Visitor-side record:

```json
{
  "id": "xtcp_...",
  "nodeId": "default",
  "name": "private-api-client",
  "role": "visitor",
  "serverName": "private-api",
  "bindAddr": "127.0.0.1",
  "bindPort": 18080,
  "secretKeySet": true,
  "enabled": true,
  "createdAt": "2026-04-30T00:00:00Z",
  "updatedAt": "2026-04-30T00:00:00Z"
}
```

Do not return `secretKey` in API responses. Follow the existing node token pattern and expose only `secretKeySet`.

## Proposed API

Keep the API local-backend only:

```text
GET    /v1/xtcp
POST   /v1/xtcp
PUT    /v1/xtcp/{ruleId}
PATCH  /v1/xtcp/{ruleId}
DELETE /v1/xtcp/{ruleId}
POST   /v1/xtcp/{ruleId}/doctor
```

Recommended request shape:

```json
{
  "nodeId": "default",
  "name": "private-api",
  "role": "server",
  "serverName": "private-api",
  "localIP": "127.0.0.1",
  "localPort": 8080,
  "secretKey": "shared-secret",
  "enabled": true
}
```

Visitor request:

```json
{
  "nodeId": "default",
  "name": "private-api-client",
  "role": "visitor",
  "serverName": "private-api",
  "bindAddr": "127.0.0.1",
  "bindPort": 18080,
  "secretKey": "shared-secret",
  "enabled": true
}
```

## Proposed CLI

Add commands under `ctl xtcp`:

```powershell
.\bin\frp-panel.exe ctl xtcp
.\bin\frp-panel.exe ctl xtcp server create private-api --local-port 8080 --secret-key <shared-secret>
.\bin\frp-panel.exe ctl xtcp visitor create private-api-client --server-name private-api --bind-port 18080 --secret-key <shared-secret>
.\bin\frp-panel.exe ctl xtcp enable private-api
.\bin\frp-panel.exe ctl xtcp disable private-api-client
.\bin\frp-panel.exe ctl xtcp delete private-api
.\bin\frp-panel.exe ctl xtcp doctor private-api-client
```

## Config Rendering Requirements

XTCP rendering must integrate with the existing multi-node renderer:

- direct TCP/UDP/HTTP rules and XTCP rules for the same `nodeId` should render into the same node-specific frpc config;
- rules for different nodes should remain in separate frpc configs and processes;
- config verification must run before persisting candidate changes;
- failed verification or reload must roll back storage and generated config, following the existing `/v1/ports` behavior.

## Doctor Requirements

XTCP diagnostics should be practical and explicit:

- verify selected node exists and has auth token;
- verify frps TCP reachability;
- verify frpc config accepts the generated XTCP proxy;
- for visitor rules, verify `bindPort` is locally available before applying;
- after apply, verify the frpc admin API reports the proxy as running when possible.

NAT traversal success may depend on real network topology. Do not promise full connectivity from static checks alone.

## Acceptance Criteria

The next implementation round is complete when:

- XTCP server and visitor rules can be created, updated, enabled, disabled, and deleted through API;
- the same operations are available through CLI;
- OpenAPI and frontend contract docs are updated;
- generated frpc config is verified by tests;
- multi-node process behavior still works;
- existing direct port tests still pass;
- at least one manual test demonstrates visitor local bind port forwarding to the service side through the configured frps node.

## Out Of Scope For The Next Round

Keep these out until XTCP basics are stable:

- full virtual LAN or room UX,
- account/user system,
- relay fallback design,
- automatic secret exchange between machines,
- server-side coordination service for pairing.
