# Next Milestone: Temporary Virtual LAN

Status: active planning target after `v0.2.0`.

## Goal

The next stage is no longer about exposing one port at a time. The product goal is to let multiple machines temporarily join the same virtual LAN with the least possible ceremony.

Primary use cases:

- temporary test environments where several machines need to see each other by virtual IP;
- short-lived remote access without deciding every port in advance;
- LAN-style game sessions where friends should feel like they are on the same private network;
- quick troubleshooting rooms that can be created, joined, inspected, and destroyed from CLI/UI.

The user experience should be closer to:

1. Create a temporary LAN room.
2. Copy one join code.
3. Other machines paste the code.
4. Everyone gets a virtual IP and can connect directly through the overlay.
5. Leaving the room stops the local process and removes the local rule.

## Recommended Engine

Use EasyTier as the first candidate engine for v0.3.

Reasons:

- It is built for virtual private networking rather than single-port forwarding.
- It supports command-line operation, which matches our backend-managed process model.
- It is cross-platform and distributed as prebuilt binaries.
- It supports NAT traversal, P2P connection attempts, relay/shared-node fallback, subnet proxying, and intelligent routing.
- It has a no-TUN/no-root mode for constrained environments, though full virtual-LAN behavior still needs TUN/admin privileges.
- It already has game-oriented ecosystem work, which is a strong signal for the LAN-game use case.

What this means for us:

- Do not build a Layer 3 VPN on top of frp/XTCP for v0.3.
- Treat EasyTier like we treat `frpc`: an external, inspectable runtime binary managed by the local backend.
- Keep frp room mode for private single-service access; add virtual LAN mode as a separate workflow.

## Alternatives Considered

- ZeroTier: mature and reliable, but it expects the ZeroTier client/service to be installed and joined through its controller model. It has `zerotier-cli`, but the install/service/control-plane assumptions are less convenient for our temporary CLI-driven flow.
- Tailscale: excellent product and CLI, but it is centered on a tailnet identity/control plane. Headscale is possible, but that turns v0.3 into an identity/control-plane project instead of a temporary LAN feature.
- NetBird: strong WireGuard-based overlay with self-hosting and CLI support, but its management service and daemon model are heavier than what we need for quick temporary rooms.
- Self-built frp/XTCP virtual LAN: not recommended for v0.3. We already proved frp XTCP for point-to-point service access, but virtual LAN needs IP routing, TUN, UDP/ICMP behavior, peer routing, diagnostics, and sometimes game broadcast handling.

## v0.3 Product Shape

Add a new "Virtual LAN Room" workflow beside direct exposure and XTCP/STCP service rooms.

Control server responsibilities:

- create a LAN room record with `lanId`, `networkName`, secret material, default CIDR, expiry, and optional join policy;
- issue a high-entropy join code;
- return join metadata to visitors without exposing stored secret hashes through list APIs;
- expire temporary rooms by default.

Local client backend responsibilities:

- store local LAN rules separately from port rules and service-room rules;
- render or generate EasyTier runtime arguments/config;
- start, stop, restart, and inspect the EasyTier process;
- report actual virtual IP, peers, route table, relay/P2P state, and recent logs;
- expose a doctor action for TUN/admin privilege, binary availability, shared-node reachability, peer discovery, and virtual-IP connectivity.

Suggested local API surface:

- `GET /v1/client/lans`: list configured local virtual LAN rules.
- `GET /v1/client/lans/status`: list local rules plus EasyTier process and peer state.
- `POST /v1/client/lans/create`: create a temporary LAN room remotely and start this machine as a member.
- `POST /v1/client/lans/join`: join from a LAN code and start this machine as a member.
- `DELETE /v1/client/lans/{lanRuleId}`: stop and remove the local LAN membership.
- `POST /v1/client/lans/{lanRuleId}/doctor`: run local diagnostics.
- `GET /v1/client/lans/{lanRuleId}/logs`: return redacted runtime logs.

Suggested control API surface:

- `POST /v1/lans`: create a temporary LAN room and return the one-time join code.
- `POST /v1/lans/join`: exchange a join code for runtime metadata.
- `GET /v1/lans/{lanId}`: fetch non-secret room metadata.
- `PATCH /v1/lans/{lanId}`: enable/disable or extend expiry for the host.
- `DELETE /v1/lans/{lanId}`: delete the room for the host.

## Runtime Defaults To Validate Tomorrow

First spike should run EasyTier manually before adding code:

1. Download the EasyTier CLI binary for Windows/Linux.
2. Start two nodes with the same network name and secret.
3. Use either our public server as the initial peer/shared node or validate EasyTier's documented shared-node mode.
4. Confirm whether full TUN mode works on Windows without manual driver steps beyond administrator privileges.
5. Confirm no-TUN mode limitations: it is useful as a fallback, but it should not be presented as full LAN mode if the local machine cannot actively initiate connections.
6. Confirm game-relevant behavior: UDP between virtual IPs first; broadcast/LAN discovery later.

Proposed defaults if the spike passes:

- Engine binary: `easytier-core` / `easytier-core.exe`, configured similarly to `frpc`.
- Workdir: `data/easytier`.
- Temporary room expiry: 24 hours.
- Virtual CIDR: allocate from an internal private range and avoid common LAN ranges when possible.
- IP assignment: start with EasyTier DHCP for fast join, then expose actual assigned IP in status. Add manual virtual IP only after collision behavior is understood.
- Server side: prefer self-hosted EasyTier shared/peer nodes over public shared nodes for reliability and privacy.

## Security And Cleanup

- LAN join codes must follow the same rule as room codes: show once, do not log, do not put in URLs, and redact from errors/logs.
- Network secrets must be stored locally only when needed to restart the EasyTier process.
- Temporary rooms should expire by default and be deletable by the host.
- Local leave must stop the EasyTier process and remove local config/log references from the active rule set.
- Logs returned through API must redact join codes and network secrets.

## Out Of Scope For First Implementation

- Automatic installation as a system service.
- Account/identity management.
- ACL UI beyond room-level secret membership.
- L2 broadcast emulation as a requirement for v0.3. Start with virtual-IP TCP/UDP connectivity; add game discovery helpers only after basic routing is stable.
- Replacing existing direct exposure or XTCP/STCP service rooms.

## References

- EasyTier GitHub: <https://github.com/EasyTier/EasyTier>
- EasyTier introduction: <https://easytier.rs/en/guide/introduction>
- EasyTier quick networking: <https://easytier.rs/en/guide/network/quick-networking>
- EasyTier no-TUN mode: <https://easytier.rs/en/guide/network/no-root>
- EasyTier full configuration options: <https://easytier.cn/en/guide/network/configurations.html>
- EasyTier game launcher: <https://easytier.rs/en/guide/gui/easytier-game.html>
- ZeroTier CLI docs: <https://docs.zerotier.com/cli>
- Tailscale CLI docs: <https://tailscale.com/docs/reference/tailscale-cli>
- NetBird CLI docs: <https://docs.netbird.io/get-started/cli>
