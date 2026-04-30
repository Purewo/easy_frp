# Client/Server XTCP Rooms

Room mode is a separated deployment model:

- Server side runs `frp-panel server` and `frps`.
- Users run `frp-panel client host` or `frp-panel client join`.
- No local HTTP backend or `ctl rooms` command is required.

## Deployment Decision

Use two frps processes during the first room-mode rollout:

- Keep the existing production frps for direct/public TCP, UDP, and HTTP exposure. It can keep its current bind port, vhost settings, auth token, and nginx wiring.
- Start a separate room frps for XTCP rooms, for example on bind port `17000`. This frps should use the HTTP plugin and should not require users to know a shared frps token.
- Point `frp-panel server --frps-addr/--frps-port` at the room frps endpoint, not the existing direct-exposure frps endpoint.
- Merge both modes onto one frps only after the room protocol, plugin authorization, and operations model are stable.

Reason: XTCP rooms require both sides to cooperate through room id, device id, and derived secret material. A separate frps lets us iterate on that authorization model without risking the existing single-port exposure workflow.

## Server

Start the control server with the public room frps endpoint:

```powershell
.\bin\frp-panel.exe server --addr :8080 --data .\data\server.json --frps-addr 149.118.158.112 --frps-port 17000
```

Configure the room frps to call the control server:

```toml
bindPort = 17000

[[httpPlugins]]
name = "frp-panel"
addr = "127.0.0.1:8080"
path = "/internal/frps/plugin"
ops = ["Login", "NewProxy", "NewUserConn"]
```

For room mode, use plugin authorization instead of distributing a shared frps token to users. See [configs/frps.room.example.toml](../configs/frps.room.example.toml).

## Client Host

The host side creates a room for a local service port and keeps frpc running in the foreground:

```powershell
.\bin\frp-panel.exe client host private-api `
  --server http://149.118.158.112:18080 `
  --frpc .\frp_0.68.1_windows_amd64\frpc.exe `
  --workdir .\data\frpc `
  --local-port 8080
```

The command prints a `roomCode`. Treat it as a secret and show or copy it immediately; later room reads do not return the code again. Send that code to the visitor.

Use `--detach` when the client must keep frpc running after the command exits. On Windows, detached mode starts frpc as an independent process so SSH or Remote Desktop session cleanup does not kill the room host.

## Client Visitor

The visitor side joins with the room code and chooses a local bind port:

```powershell
.\bin\frp-panel.exe client join <roomCode> `
  --server http://149.118.158.112:18080 `
  --frpc .\frp_0.68.1_windows_amd64\frpc.exe `
  --workdir .\data\frpc `
  --bind-port 18080
```

Traffic to `127.0.0.1:18080` on the visitor machine is forwarded to the host service.

## Frontend Integration Contract

Room mode has two separate integration surfaces:

- The control-server API stores global room records, room codes, device credentials, and server-side room enabled state.
- The local client backend stores this machine's room rules, writes frpc config, starts/reloads frpc, and reports local process/NAT state.

Do not wire room mode through the local direct-port `/v1/ports` API. That API remains for single-client direct exposure and is documented in [local-client-api.md](local-client-api.md).

The local backend does expose room-specific endpoints under `/v1/client/rooms...` for the desktop/frontend shell. These endpoints store local room rules, render frpc configs, start or reload the matching frpc process, and report local process/NAT status. The public control server remains the source of room records and room codes.

Current development frontend wiring:

| Frontend client | Base URL | Use for |
| --- | --- | --- |
| `controlApi` | `http://149.118.158.112:18080` | Global room list, room record lookup, server-side room management. |
| `clientApi` | `http://127.0.0.1:7410` | Local host/join actions, local room process state, local NAT diagnostics. |

Do not start or target a local `frp-panel server` on `127.0.0.1:8080` for the current dev UI. That process is only needed for private control-server deployments. The checked-in web client already uses `CONTROL_SERVER_URL = "http://149.118.158.112:18080"` for control APIs and keeps `clientApi` on `http://127.0.0.1:7410`.

Important data boundary:

- `GET http://149.118.158.112:18080/v1/rooms` returns all server-side rooms, including rooms created by other clients or previous tests.
- `GET http://127.0.0.1:7410/v1/client/rooms/status` returns only room rules created or joined on this local machine through the local backend.
- It is normal for the remote room list to contain records while the local status list is empty. That only means this client has no local room process configured yet.

Recommended frontend flow:

1. List remote room records with `controlApi.get("/v1/rooms")`.
2. Create a host tunnel from the local backend with `clientApi.post("/v1/client/rooms/host", ...)`; omit `serverBaseURL` unless the user explicitly targets a private control server.
3. Join a room from the local backend with `clientApi.post("/v1/client/rooms/join", ...)`; the same `roomCode` must be handed to the visitor, and both sides must use the same `tunnelProtocol`.
4. Render local running state from `clientApi.get("/v1/client/rooms/status")`, not from the remote room list.
5. For XTCP troubleshooting, call `GET /v1/client/network/interfaces`, `POST /v1/client/xtcp/nathole/discover`, and `POST /v1/client/rooms/{roomRuleId}/doctor` on the local backend.

## Control Server API

Base URL is the public control server. The current built-in development target is `http://149.118.158.112:18080`. The OpenAPI source is [openapi/openapi.yaml](../openapi/openapi.yaml), and generated frontend types live in [web/src/api/types.ts](../web/src/api/types.ts).

All room API errors use:

```json
{
  "error": "message"
}
```

Expected room error statuses are `400` for invalid input or disabled rooms, `401` for invalid room code or room device auth, and `404` for missing rooms.

### `POST /v1/rooms`

Create a host room and issue the host device credentials.

Request:

```json
{
  "name": "private-api",
  "deviceName": "alice-laptop"
}
```

Response `201`:

```json
{
  "room": {
    "id": "room_abc",
    "name": "private-api",
    "serverName": "room.room_abc.xtcp",
    "hostDeviceId": "rdev_host",
    "frpsAddr": "rooms.frps.example.com",
    "frpsPort": 17000,
    "enabled": true,
    "memberCount": 1,
    "createdAt": "2026-04-30T10:00:00Z",
    "updatedAt": "2026-04-30T10:00:00Z"
  },
  "roomCode": "room_abc.secret",
  "device": {
    "id": "rdev_host",
    "roomId": "room_abc",
    "name": "alice-laptop",
    "role": "host",
    "createdAt": "2026-04-30T10:00:00Z",
    "lastSeenAt": "2026-04-30T10:00:00Z"
  },
  "deviceToken": "host-device-token"
}
```

Frontend rules:

- Show `roomCode` once and provide a copy action.
- Store `device.id` and `deviceToken` only in the local desktop/client store if the UI needs to manage the room later.
- Do not send `roomCode` or `deviceToken` to analytics, logs, URLs, or crash reports.

### `POST /v1/rooms/join`

Register a visitor device from a room code.

Request:

```json
{
  "roomCode": "room_abc.secret",
  "deviceName": "bob-desktop"
}
```

Response `200` has the same shape as create, except it has no `roomCode`, and `device.role` is `visitor`.

### `GET /v1/rooms`

List room summaries for the control UI.

Response `200`:

```json
[
  {
    "id": "room_abc",
    "name": "private-api",
    "serverName": "room.room_abc.xtcp",
    "hostDeviceId": "rdev_host",
    "frpsAddr": "rooms.frps.example.com",
    "frpsPort": 17000,
    "enabled": true,
    "memberCount": 2,
    "createdAt": "2026-04-30T10:00:00Z",
    "updatedAt": "2026-04-30T10:05:00Z"
  }
]
```

### `GET /v1/rooms/{roomId}`

Read one room summary. This is useful for a room detail page after create or join.

### `PATCH /v1/rooms/{roomId}`

Enable or disable a room. Only the host device can manage room state.

Headers:

```http
X-Room-ID: room_abc
X-Room-Device-ID: rdev_host
X-Room-Device-Token: host-device-token
```

Request:

```json
{
  "enabled": false
}
```

Response `200` is `RoomView`.

The server also accepts `Authorization: Bearer <deviceToken>` as a token fallback, but new frontend code should send `X-Room-Device-Token` explicitly.

### `DELETE /v1/rooms/{roomId}`

Delete a room and all registered room devices. Use the same host-device headers as `PATCH`. Response is `204` with an empty body.

## Desktop/Client Bridge

The frontend should treat CLI execution as the source of local tunnel state. The control API only creates room and device records; traffic does not flow until local `frpc` is started.

Host flow:

```powershell
.\bin\frp-panel.exe client host private-api `
  --server http://149.118.158.112:18080 `
  --frpc .\frp_0.68.1_windows_amd64\frpc.exe `
  --workdir .\data\frpc `
  --device-name alice-laptop `
  --local-ip 127.0.0.1 `
  --local-port 8080 `
  --json
```

Visitor flow:

```powershell
.\bin\frp-panel.exe client join <roomCode> `
  --server http://149.118.158.112:18080 `
  --frpc .\frp_0.68.1_windows_amd64\frpc.exe `
  --workdir .\data\frpc `
  --device-name bob-desktop `
  --bind-addr 127.0.0.1 `
  --bind-port 18080 `
  --json
```

Local room list:

```powershell
.\bin\frp-panel.exe client rooms `
  --server http://149.118.158.112:18080 `
  --frpc .\frp_0.68.1_windows_amd64\frpc.exe `
  --workdir .\data\frpc `
  --json
```

CLI JSON output is `RoomRuleView`:

```json
{
  "id": "room-rule-id",
  "roomId": "room_abc",
  "roomCode": "room_abc.secret",
  "name": "private-api",
  "role": "host",
  "serverName": "room.room_abc.xtcp",
  "serverAddr": "rooms.frps.example.com",
  "serverPort": 17000,
  "deviceId": "rdev_host",
  "deviceTokenSet": true,
  "secretKeySet": true,
  "localIP": "127.0.0.1",
  "localPort": 8080,
  "enabled": true,
  "createdAt": "2026-04-30T10:00:00Z",
  "updatedAt": "2026-04-30T10:00:00Z"
}
```

For visitors, `role` is `visitor`, `roomCode` is omitted, and `bindAddr` plus `bindPort` are present. The UI should display the visitor target as `bindAddr:bindPort`, for example `127.0.0.1:18080`.

Use `--detach` only when the product wants frpc to survive after the wrapper command exits. Without `--detach`, keep the process attached and show a running state until the process exits or the user stops it.

## Local Backend Room API

The desktop/frontend shell should use the local client backend at `http://127.0.0.1:7410` for local room process management:

- `GET /v1/client/rooms`: list local room rules.
- `GET /v1/client/rooms/status`: list local room rules plus their frpc process state.
- `POST /v1/client/rooms/host`: create a control-server room, save a local host rule, render config, and start/reload frpc.
- `POST /v1/client/rooms/join`: join from `roomCode`, save a local visitor rule, render config, and start/reload frpc.
- `PATCH /v1/client/rooms/{roomRuleId}`: enable or disable a local room rule.
- `DELETE /v1/client/rooms/{roomRuleId}`: delete a local room rule and reapply frpc.
- `POST /v1/client/rooms/{roomRuleId}/doctor`: validate the room rule, frps TCP reachability, visitor bind port, and XTCP NAT discovery.
- `GET /v1/client/network/interfaces`: list local IPv4 interfaces for diagnostics.
- `POST /v1/client/xtcp/nathole/discover`: run `frpc nathole discover` with optional `stunServer` and diagnostic-only `localAddr`.

Host request:

```json
{
  "name": "private-api",
  "deviceName": "alice-laptop",
  "tunnelProtocol": "xtcp",
  "natHoleStunServer": "stun.easyvoip.com:3478",
  "localIP": "127.0.0.1",
  "localPort": 8080
}
```

Visitor request:

```json
{
  "roomCode": "room_abc.secret",
  "deviceName": "bob-desktop",
  "tunnelProtocol": "xtcp",
  "bindAddr": "127.0.0.1",
  "bindPort": 18080
}
```

Use `"tunnelProtocol": "stcp"` as the reliable fallback when XTCP NAT discovery is unstable. Both host and visitor must use the same tunnel protocol for the same room handoff.

`serverBaseURL` is optional for both host and visitor requests. Leave it blank in the normal frontend path so the local backend uses its built-in control server, currently `http://149.118.158.112:18080`. Only send `serverBaseURL` for an advanced/private-server override.

NAT diagnostic request:

```json
{
  "stunServer": "stun.easyvoip.com:3478",
  "localAddr": "10.7.24.208:0"
}
```

`localAddr` is diagnostic-only. frp 0.68.1 supports it for `frpc nathole discover`, but not as a persistent room config field. Persisted room config can set `natHoleStunServer`.

## Frontend State Model

- `RoomView.enabled` is the server-side switch. Disabling a room makes future room authorization fail; the client UI should still stop local frpc when the user wants the local process gone.
- `RoomView.memberCount` counts registered devices, not active P2P sessions.
- `RoomDeviceView.lastSeenAt` advances when a device authenticates through room-management APIs. It is not a live tunnel heartbeat.
- XTCP/STCP connection health comes from `/v1/client/rooms/status`, `/v1/client/rooms/{roomRuleId}/doctor`, and local frpc logs, not from `GET /v1/rooms`.
- In hard NAT, connection establishment can take time. Show a pending state while frpc is running and only fail when the process exits or logs an error.

## Frontend Validation

- `name` and `deviceName`: trim whitespace, 1 to 80 characters.
- `localIP`: default `127.0.0.1`; must be an IP address or `localhost`.
- `localPort` and `bindPort`: required, 1 to 65535.
- `bindAddr`: default `127.0.0.1`. Avoid `0.0.0.0` unless the user explicitly wants other machines on the LAN to access the visitor port.
- `roomCode`: required for visitors; treat it as a secret input.
- `frpc` path and `workdir`: required for desktop/client integration. Validate existence before starting the CLI when possible.

Recommended screens:

- Room list: name, enabled state, member count, room frps endpoint, created/updated time.
- Create host room: room name, device name, local IP, local port, foreground or detached mode.
- Room code handoff: copy-only secret display after create.
- Join room: room code, device name, bind address, bind port.
- Active local room: role, local endpoint, frps endpoint, process state, recent frpc logs.
- Room management: disable, enable, delete. Show these actions only when host device credentials are available.

## Security Model

The room code is high entropy and contains the room id plus secret material. The client derives the XTCP `secretKey` locally from the code. The server stores only a hash of the room code and device token hashes.

frp 0.68.1 does not support metadata on individual visitor records, so the client renders each enabled room rule into its own frpc config with global room metadata.

Backend hardening rules:

- Host and visitor requests are locally validated before the backend calls the remote control server. This prevents obvious invalid ports, invalid protocols, bad STUN values, or occupied visitor bind ports from creating remote room/device records.
- If host room creation succeeds remotely but local config persistence, verification, or frpc apply fails, the local backend makes a best-effort authenticated delete call to remove the remote room.
- Room codes are returned only by the immediate host-create response. List/status APIs do not return room codes, device tokens, or secret keys.
- Text logs returned by the local backend are redacted for room codes, device tokens, auth tokens, admin passwords, and secret keys.
- Detached frpc status checks the OS process state instead of trusting the original PID forever.

## Hard NAT Test Notes

Manual testing on 2026-04-30 succeeded with a Windows host behind hard NAT after direct SSH to the host timed out. The room visitor bound a local port and reached the host's OpenSSH service and a simple HTTP site through XTCP over QUIC.

Observed frpc NAT-hole logs included `HardNAT`, `BehaviorPortChanged`, `Mode:2`, `SendRandomPorts:1000`, and `establishing nat hole connection successful`. This validates the room flow as a real NAT traversal path, not just a LAN route.

Manual testing on 2026-05-01 also succeeded with a local web page served from the host endpoint `127.0.0.1:90` and a Windows visitor binding `127.0.0.1:19090`. After WiFi was disconnected, direct SSH to the Windows machine timed out, while the visitor request eventually returned HTTP `200` and matched the expected page marker. Host and visitor logs showed `HardNAT`, `BehaviorPortChanged`, `quic`, and `establishing nat hole connection successful`. The room code used for the test is intentionally not recorded here.
