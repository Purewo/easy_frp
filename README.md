# frp UI Backend

Go backend for a graphical frp control plane.

It contains two HTTP services:

- Public control server: group/device registry, node registry, public exposure allocation, private route metadata, and frps HTTP plugin checks.
- Windows local client backend: localhost API for TCP/UDP port exposure and HTTP web exposure, frpc config rendering, frpc process control, and local rule persistence.

Frontend work for the current direct-port UI should use [docs/local-client-api.md](docs/local-client-api.md) and [openapi/local-client.yaml](openapi/local-client.yaml). The older combined API contract remains in [openapi/openapi.yaml](openapi/openapi.yaml) for legacy control-server flows.

Current release: [v0.2.0](docs/releases/v0.2.0.md). XTCP room mode is documented in [docs/client-server-rooms.md](docs/client-server-rooms.md).

## Build

```powershell
go test ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go build -o bin\frp-panel.exe .\cmd\frp-panel
```

## Run public control server

```powershell
.\bin\frp-panel.exe server --addr :8080 --data .\data\server.json --frps-addr rooms.frps.example.com --frps-port 17000
```

For XTCP room mode, run this control server on the same server side as `frps`. During the first rollout, keep the existing production frps for direct/public exposure and start a second room-only frps, for example on port `17000` with [configs/frps.room.example.toml](configs/frps.room.example.toml). Point `--frps-addr` and `--frps-port` at that room frps. Merge the two frps processes only after the room protocol and plugin authorization are stable.

The room frps should call the HTTP plugin endpoint below so the control server can authorize room clients by metadata instead of giving users a shared frps token.

## Run Windows local client backend

Use `.env.example` as the list of local environment variables. Keep real tokens in your shell or local `.env`; `.env` files are ignored by git.

```powershell
$env:FRP_PANEL_FRPS_TOKEN='<frps auth token>'
.\bin\frp-panel.exe client --addr 127.0.0.1:7410 --data .\data\client.json --frpc .\frp_0.68.1_windows_amd64\frpc.exe --workdir .\data\frpc --server http://149.118.158.112:18080 --frps-host 149.118.158.112 --frps-port 7000 --web-base-domain ma1.gameuniverse.top
```

The local backend seeds a `default` frps node, writes frpc config under the configured work directory, verifies candidate config with `frpc verify -c <config>`, then starts frpc or uses `frpc reload -c <config>` when port rules change. More frps nodes can be added later through the API, CLI, or UI. In direct-port mode, enabled rules are grouped by `nodeId`; each active node gets its own config file, admin port, frpc process, and log file.

Room mode uses two API surfaces: the remote control server at `http://149.118.158.112:18080` for room records, and the local backend at `http://127.0.0.1:7410` for this machine's room rules and frpc lifecycle. See [docs/client-server-rooms.md](docs/client-server-rooms.md) for the frontend wiring contract.

Direct exposure APIs:

- `GET /v1/nodes`
- `POST /v1/nodes`
- `PUT /v1/nodes/{nodeId}`
- `POST /v1/nodes/{nodeId}/doctor`
- `DELETE /v1/nodes/{nodeId}`
- `GET /v1/ports`
- `POST /v1/ports`
- `PUT /v1/ports/{portId}`
- `PATCH /v1/ports/{portId}`
- `DELETE /v1/ports/{portId}`

See [docs/local-client-api.md](docs/local-client-api.md) for request/response examples, validation rules, and UI guidance.

## Local CLI

The `ctl` command talks to the running local backend, defaulting to `http://127.0.0.1:7410`. It uses the same backend validation, persistence, config rendering, and frpc reload path as the UI.

List nodes and rules:

```powershell
.\bin\frp-panel.exe ctl nodes
.\bin\frp-panel.exe ctl ports
.\bin\frp-panel.exe ctl status
```

Manage frps nodes:

```powershell
.\bin\frp-panel.exe ctl nodes create --id backup --server-addr 149.118.158.112 --frps-port 7000 --token <frps auth token> --web-base-domain ma1.gameuniverse.top
.\bin\frp-panel.exe ctl nodes update backup --name "Backup frps 2"
.\bin\frp-panel.exe ctl nodes doctor backup
.\bin\frp-panel.exe ctl nodes delete backup
```

Expose a local web app through HTTPS:

```powershell
.\bin\frp-panel.exe ctl expose cyberstream 18089
```

That shorthand creates an HTTP rule named `cyberstream`, maps `127.0.0.1:18089`, and publishes `https://cyberstream.ma1.gameuniverse.top`.

Create explicit HTTP and TCP rules:

```powershell
.\bin\frp-panel.exe ctl ports create --protocol http --name blog --local-port 3000 --subdomain blog
.\bin\frp-panel.exe ctl ports create --protocol tcp --local-port 8080 --remote-port 18080
.\bin\frp-panel.exe ctl ports create --node-id backup --protocol http --local-port 5173 --subdomain dev
```

Operate rules by id, name, domain, or subdomain:

```powershell
.\bin\frp-panel.exe ctl disable cyberstream
.\bin\frp-panel.exe ctl enable cyberstream.ma1.gameuniverse.top
.\bin\frp-panel.exe ctl delete cyberstream
.\bin\frp-panel.exe ctl reload
.\bin\frp-panel.exe ctl logs --tail 80
```

Use `--json` for machine-readable `ctl` output and `--api` or `FRP_PANEL_API` when the local backend is not on the default address.

XTCP room mode:

```powershell
.\bin\frp-panel.exe client host private-api --server http://149.118.158.112:18080 --local-port 8080 --frpc .\frp_0.68.1_windows_amd64\frpc.exe --workdir .\data\frpc
.\bin\frp-panel.exe client join <roomCode> --server http://149.118.158.112:18080 --bind-port 18080 --frpc .\frp_0.68.1_windows_amd64\frpc.exe --workdir .\data\frpc
.\bin\frp-panel.exe client rooms --data .\data\client.json
```

The host command prints a one-time high-entropy `roomCode`, starts frpc, and keeps the client process in the foreground. The visitor command uses that code plus its own local bind port. The room code is used locally to derive the XTCP `secretKey`; the control server stores only a hash of the code.

Add `--detach` when the room process should keep running after the CLI exits. Detached mode starts frpc independently, which is important for Windows hosts launched over SSH or Remote Desktop session cleanup.

Create a TCP exposure:

```powershell
Invoke-RestMethod http://127.0.0.1:7410/v1/ports -Method Post -ContentType 'application/json' -Body '{
  "nodeId": "default",
  "protocol": "tcp",
  "localIP": "127.0.0.1",
  "localPort": 8080,
  "remotePort": 18080
}'
```

Create an HTTPS web exposure through the server wildcard domain:

```powershell
Invoke-RestMethod http://127.0.0.1:7410/v1/ports -Method Post -ContentType 'application/json' -Body '{
  "nodeId": "default",
  "protocol": "http",
  "localIP": "127.0.0.1",
  "localPort": 3000,
  "subdomain": "blog"
}'
```

This generates an frpc HTTP proxy for `blog.ma1.gameuniverse.top`; the public URL is `https://blog.ma1.gameuniverse.top`. HTTP rules automatically rewrite the upstream Host header to the local host, so Vite and similar dev servers usually do not need per-domain `allowedHosts` changes.

## frps plugin

Configure frps HTTP plugin to call:

```text
POST http://<control-server>/internal/frps/plugin
```

Local example with the downloaded frp package:

```powershell
.\frp_0.68.1_windows_amd64\frps.exe -c .\configs\frps.panel.example.toml
.\frp_0.68.1_windows_amd64\frps.exe -c .\configs\frps.room.example.toml
```

The plugin expects frpc metadata fields:

- `group_id`
- `device_id`
- `device_token`
- `exposure_id` for proxy creation
- `room_id`
- `room_device_id`
- `room_device_token`
- `room_role` for XTCP room clients

## frp compatibility check

After downloading frp, run this optional test to verify generated client config still matches the installed frpc version:

```powershell
$env:FRPC_EXE='G:\AI\AI_private\Codex_projects\frp\frp_0.68.1_windows_amd64\frpc.exe'
go test ./internal/frp -run TestRenderedClientConfigAcceptedByFrpc -v
```

## frp end-to-end check

Run this optional integration test to start a temporary control server, frps, frpc, and local TCP service, then verify a public TCP exposure can carry traffic:

```powershell
$env:FRPC_EXE='G:\AI\AI_private\Codex_projects\frp\frp_0.68.1_windows_amd64\frpc.exe'
$env:FRPS_EXE='G:\AI\AI_private\Codex_projects\frp\frp_0.68.1_windows_amd64\frps.exe'
go test ./internal/e2e -run TestFrpPublicTCPExposureEndToEnd -v
```
