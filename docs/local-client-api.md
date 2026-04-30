# Local Client API for Frontend

This document is the frontend contract for the current product direction: expose local TCP/UDP ports and HTTP web services through real frps nodes. The frontend should use the local backend only:

```text
http://127.0.0.1:7410
```

The OpenAPI source for this UI flow is [openapi/local-client.yaml](../openapi/local-client.yaml). The older control-server/group/room APIs are legacy for now and should not drive the new UI.

## Product Flow

1. Start the local backend with a frps token.
2. Read, add, edit, or delete frps nodes with `/v1/nodes`.
3. Read current rules with `GET /v1/ports`.
4. Create, edit, enable/disable, or delete an exposure rule.
5. The backend immediately verifies and applies frpc config. The UI does not need a separate "save config" step.

Multi-node behavior: the backend stores multiple frps nodes and runs one local frpc process per active `nodeId`. Enabled rules can therefore run on different frps servers at the same time. Each node gets its own generated config file, admin port, process state, and log file.

## Start Backend

```powershell
$env:FRP_PANEL_FRPS_TOKEN='<frps auth token>'
.\bin\frp-panel.exe client `
  --addr 127.0.0.1:7410 `
  --data .\data\client.json `
  --frpc .\frp_0.68.1_windows_amd64\frpc.exe `
  --workdir .\data\frpc `
  --frps-host 149.118.158.112 `
  --frps-port 7000 `
  --web-base-domain ma1.gameuniverse.top `
  --web-scheme https `
  --vhost-http-port 8080
```

The backend seeds a default node:

```json
{
  "id": "default",
  "name": "Default frps",
  "serverAddr": "149.118.158.112",
  "frpsPort": 7000,
  "authMethod": "token",
  "authTokenSet": true,
  "webBaseDomain": "ma1.gameuniverse.top",
  "webScheme": "https",
  "vhostHTTPPort": 8080
}
```

`authToken` is never returned by API responses.

Additional nodes can be created through API or CLI. A node represents one frps server, including its connection port, token, optional HTTPS wildcard base domain, and optional TCP/UDP remote port allow-list.

## CLI Contract

The local CLI is available through the same binary:

```powershell
.\bin\frp-panel.exe ctl [--api http://127.0.0.1:7410] <command>
```

It calls this local API instead of editing config files directly, so CLI and UI behavior stay identical.

Common commands:

```powershell
.\bin\frp-panel.exe ctl nodes
.\bin\frp-panel.exe ctl ports
.\bin\frp-panel.exe ctl status
.\bin\frp-panel.exe ctl reload
.\bin\frp-panel.exe ctl logs --tail 80
```

Create, update, and delete frps nodes:

```powershell
.\bin\frp-panel.exe ctl nodes create `
  --id backup `
  --name "Backup frps" `
  --server-addr 149.118.158.112 `
  --frps-port 7000 `
  --token <frps auth token> `
  --web-base-domain ma1.gameuniverse.top `
  --web-scheme https `
  --vhost-http-port 8080

.\bin\frp-panel.exe ctl nodes update backup --name "Backup frps 2"
.\bin\frp-panel.exe ctl nodes doctor backup
.\bin\frp-panel.exe ctl nodes delete backup
```

Create a rule on a non-default node:

```powershell
.\bin\frp-panel.exe ctl ports create --node-id backup --protocol http --local-port 5173 --subdomain dev
```

Fast web exposure shorthand:

```powershell
.\bin\frp-panel.exe ctl expose cyberstream 18089
```

Equivalent explicit command:

```powershell
.\bin\frp-panel.exe ctl ports create `
  --protocol http `
  --name cyberstream `
  --local-ip 127.0.0.1 `
  --local-port 18089 `
  --subdomain cyberstream
```

TCP/UDP exposure:

```powershell
.\bin\frp-panel.exe ctl ports create --protocol tcp --local-port 8080 --remote-port 18080
```

Rule selectors for `delete`, `enable`, and `disable` accept any unique rule `id`, `name`, `domain`, or `subdomain`:

```powershell
.\bin\frp-panel.exe ctl disable cyberstream
.\bin\frp-panel.exe ctl enable cyberstream.ma1.gameuniverse.top
.\bin\frp-panel.exe ctl delete port_0123abcd4567ef89
```

Use `--json` for machine-readable output:

```powershell
.\bin\frp-panel.exe ctl --json ports
```

## API Endpoints

### `GET /v1/nodes`

Returns local frps nodes. `authToken` is not returned; use `authTokenSet` to show whether the node can start frpc.

### `POST /v1/nodes`

Creates a new local frps node. This only stores node configuration; frpc is not restarted until an enabled rule uses the node.

```json
{
  "id": "backup",
  "name": "Backup frps",
  "serverAddr": "149.118.158.112",
  "frpsPort": 7000,
  "authMethod": "token",
  "authToken": "your-frps-token",
  "webBaseDomain": "ma1.gameuniverse.top",
  "webScheme": "https",
  "vhostHTTPPort": 8080,
  "allowPorts": [
    { "from": 18000, "to": 18999 }
  ]
}
```

Response: `201` with the created `NodeView`.

### `PUT /v1/nodes/{nodeId}`

Replaces a node. Send all editable fields. `authToken` is optional on update: omit it to keep the existing token, send it to replace the token, or set `clearAuthToken: true` to remove it.

If the node has enabled rules, the backend verifies and reloads frpc with the new node config immediately. If verification or reload fails, the node update is rolled back.

```json
{
  "name": "Backup frps 2",
  "serverAddr": "149.118.158.112",
  "frpsPort": 7000,
  "authMethod": "token",
  "webBaseDomain": "ma1.gameuniverse.top",
  "webScheme": "https",
  "vhostHTTPPort": 8080,
  "allowPorts": []
}
```

### `DELETE /v1/nodes/{nodeId}`

Deletes a node. The `default` node cannot be deleted, and a node cannot be deleted while any port rule still references it, even if that rule is disabled.

Response: `204` with an empty body.

### `POST /v1/nodes/{nodeId}/doctor`

Runs a read-only diagnostic for one frps node. The backend checks:

- node config validation
- auth token presence
- TCP connectivity to `serverAddr:frpsPort`
- temporary frpc login to validate server/token compatibility
- wildcard DNS for `webBaseDomain`, when configured
- public Web entrypoint for a random diagnostic subdomain, when configured

The temporary frpc login uses a no-proxy config and is stopped automatically; it does not create or modify exposure rules.

```json
{
  "overall": "pass",
  "testedDomain": "doctor-ab12cd34.ma1.gameuniverse.top",
  "node": {
    "id": "default",
    "name": "Default frps",
    "serverAddr": "149.118.158.112",
    "frpsPort": 7000,
    "authMethod": "token",
    "authTokenSet": true,
    "webBaseDomain": "ma1.gameuniverse.top",
    "webScheme": "https",
    "vhostHTTPPort": 8080
  },
  "checks": [
    {
      "id": "frpc-login",
      "name": "frpc 登录验证",
      "status": "pass",
      "message": "临时 frpc 登录 frps 成功",
      "durationMs": 320
    }
  ]
}
```

Check status values:

- `pass`: usable
- `warn`: usable but suspicious, for example DNS points to a CDN address instead of `serverAddr`
- `fail`: blocking issue
- `skipped`: not applicable, for example web checks when `webBaseDomain` is empty

### `GET /v1/ports`

Returns all local exposure rules. A TCP/UDP rule has `remotePort`; an HTTP web rule has `subdomain` and `domain`.

```json
[
  {
    "id": "port_0123abcd4567ef89",
    "nodeId": "default",
    "name": "web-tcp",
    "protocol": "tcp",
    "localIP": "127.0.0.1",
    "localPort": 8080,
    "remotePort": 18080,
    "enabled": true,
    "createdAt": "2026-04-30T10:00:00Z",
    "updatedAt": "2026-04-30T10:00:00Z"
  },
  {
    "id": "port_abcdef0123456789",
    "nodeId": "default",
    "name": "blog",
    "protocol": "http",
    "localIP": "127.0.0.1",
    "localPort": 3000,
    "subdomain": "blog",
    "domain": "blog.ma1.gameuniverse.top",
    "enabled": true,
    "createdAt": "2026-04-30T10:00:00Z",
    "updatedAt": "2026-04-30T10:00:00Z"
  }
]
```

### `POST /v1/ports`

Creates an exposure and applies it immediately.

TCP/UDP request:

```json
{
  "nodeId": "default",
  "name": "web-tcp",
  "protocol": "tcp",
  "localIP": "127.0.0.1",
  "localPort": 8080,
  "remotePort": 18080,
  "enabled": true
}
```

HTTP web request using a subdomain:

```json
{
  "nodeId": "default",
  "name": "blog",
  "protocol": "http",
  "localIP": "127.0.0.1",
  "localPort": 3000,
  "subdomain": "blog",
  "enabled": true
}
```

HTTP web request using a full domain:

```json
{
  "nodeId": "default",
  "protocol": "http",
  "localIP": "127.0.0.1",
  "localPort": 5173,
  "domain": "dev.ma1.gameuniverse.top"
}
```

Required fields:

- All protocols: `protocol`, `localPort`.
- TCP/UDP: `remotePort`.
- HTTP: `subdomain` or `domain` is recommended. If omitted, the backend derives a subdomain from `name` or the generated rule id.

Defaults:

- `nodeId`: `default`
- `localIP`: `127.0.0.1`
- `enabled`: `true`
- TCP/UDP `name`: `<protocol>-<remotePort>`
- HTTP `name`: `<subdomain>` or `<domain>`

Response: `201` with the created `PortRule`.

### `PUT /v1/ports/{portId}`

Replaces a rule and applies the new frpc config immediately. Send all editable fields for the selected protocol.

### `PATCH /v1/ports/{portId}`

Enable or disable a rule.

```json
{
  "enabled": false
}
```

Disabled rules stay in the list but are omitted from generated `frpc.toml`.

### `DELETE /v1/ports/{portId}`

Deletes a rule and applies the new frpc config immediately.

Response: `204` with an empty body.

### `GET /v1/frpc/status`

Returns the local frpc process state. In direct multi-node mode, top-level `running` is true when at least one node process is running, and `nodes` contains the per-node process list.

```json
{
  "running": true,
  "configPath": ".\\data\\frpc\\frpc.toml",
  "nodes": [
    {
      "nodeId": "default",
      "running": true,
      "pid": 12345,
      "configPath": ".\\data\\frpc\\frpc.toml",
      "logPath": ".\\data\\frpc\\frpc.log"
    },
    {
      "nodeId": "backup",
      "running": true,
      "pid": 12346,
      "configPath": ".\\data\\frpc\\frpc.backup.toml",
      "logPath": ".\\data\\frpc\\frpc.backup.log"
    }
  ]
}
```

### `POST /v1/frpc/reload`

Manually rewrites and reloads the current config. The normal create/edit/delete flow already applies changes, so this is mainly a recovery/debug action.

### `GET /v1/logs`

Returns recent frpc logs as `text/plain`. When multiple node processes are active, the response contains per-node sections.

## Generated frpc Shape

Default node config is written to:

```text
<workdir>\frpc.toml
```

Non-default node configs are written to:

```text
<workdir>\frpc.<nodeId>.toml
```

The default node keeps `webServer.port = 7400` unless configured otherwise. Additional active nodes receive deterministic separate admin ports so multiple frpc processes can run at the same time.

TCP/UDP:

```toml
[[proxies]]
name = "port.port_0123abcd4567ef89"
type = "tcp"
localIP = "127.0.0.1"
localPort = 8080
remotePort = 18080
```

HTTP web service:

```toml
[[proxies]]
name = "web.port_abcdef0123456789"
type = "http"
localIP = "127.0.0.1"
localPort = 3000
customDomains = ["blog.ma1.gameuniverse.top"]
hostHeaderRewrite = "127.0.0.1"
```

The public HTTPS URL is:

```text
https://blog.ma1.gameuniverse.top
```

For HTTP rules, the backend automatically writes `hostHeaderRewrite` with the local host. This prevents common local dev servers, such as Vite, from rejecting arbitrary wildcard domains through Host header validation. Users normally do not need to edit `vite.config.js` or add every new public subdomain to `server.allowedHosts`.

## Validation and Errors

All JSON errors use:

```json
{
  "error": "message"
}
```

Important validation rules:

- Node `id` must be 2-64 chars and contain only letters, numbers, `_`, or `-`.
- Node `serverAddr` must be a host or IP without scheme, path, or port.
- Node `authMethod` must be `token`.
- Node `webScheme` must be `http` or `https`.
- Node `webBaseDomain`, when set, must be a bare domain without scheme or port.
- Node `allowPorts`, when set, restricts TCP/UDP `remotePort` creation on that node.
- `protocol` must be `tcp`, `udp`, or `http`.
- `localPort` must be `1-65535`.
- TCP/UDP `remotePort` must be `1024-65535`.
- HTTP `domain` must be under `node.webBaseDomain`, currently `ma1.gameuniverse.top`.
- HTTP `subdomain` must be one DNS label: lowercase letters, numbers, and `-`.
- `localIP` must be an IP address or `localhost`.
- `nodeId` must exist.
- `node auth token is not configured` means the backend was started without `FRP_PANEL_FRPS_TOKEN` or `--frps-token`.
- Duplicate TCP/UDP `nodeId + protocol + remotePort` returns `409`.
- Duplicate HTTP `nodeId + domain` returns `409`.
- Enabled rules can span multiple `nodeId` values. The backend generates and applies one frpc config per active node.
- If `frpc verify` or reload/start fails, the backend rolls back local data and config, then returns an error.

## UI Recommendations

- Main table columns: name, protocol, local address, public address, enabled, status/action.
- Provide a node management page or drawer for frps server add/edit/delete. Show `serverAddr:frpsPort`, wildcard web domain, token state, and how many rules reference each node.
- Provide a "doctor" action on each node and render `POST /v1/nodes/{nodeId}/doctor` checks directly.
- Disable delete for the `default` node and nodes that are still referenced by rules.
- In rule creation, node selection should be explicit. The status page should render `/v1/frpc/status.nodes[]` when more than one node process exists.
- Create form protocol choices:
  - TCP/UDP: localIP, localPort, remotePort.
  - Web HTTPS: localIP, localPort, subdomain or full domain.
- Show TCP/UDP public address as `<node.serverAddr>:<remotePort>`.
- Show HTTP public address as `<node.webScheme>://<domain>`.
- Show `authTokenSet=false` as a blocking setup warning before allowing rule creation.
- After any mutation, refetch `/v1/ports` and `/v1/frpc/status`.
