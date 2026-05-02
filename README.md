# Valheim Plugin

`proxygw-valheim` is an external plugin module that provides a frontend handler for
the game [Valheim](https://www.valheimgame.com/)

## Plugin Setup

Add the plugin to the `proxygw` daemon build in the main `proxygw` repository:

1. Add this module as a dependency:

```sh
go get github.com/UselessMnemonic/proxygw-valheim@latest
```

2. Register the plugin in `plugin.yaml`:

```yaml
plugins:
  github.com/UselessMnemonic/proxygw-valheim: valheim
```

3. Regenerate the plugin import file and rebuild the daemon:

```sh
make proxygw
```

The plugin registers under the module path
`github.com/UselessMnemonic/proxygw-valheim` and uses the `valheim`
namespace, so frontend kinds are referenced as `valheim:...`.

## Exported Kinds

Frontends:

- `server`: Listens on Valheim's gameplay UDP port and warms the target when a
  player sends traffic.
- `status`: Listens on Valheim's Steam query UDP port and answers A2S status
  probes without warming the target.

There is no plugin-level configuration for this plugin.

## Valheim Setup

Valheim dedicated servers use UDP. In direct Steam mode, the server listens on
the configured gameplay port and on the next port for Steam Query/A2S. With the
default `-port 2456`, gameplay is `2456/udp` and A2S is `2457/udp`.

Example:

```yaml
frontends:
  - name: server-frontend
    kind: valheim:server
    target: real-server
    protocol: udp
    listen: [::]:2456
    flow_timeout: 30s

  - name: status-frontend
    kind: valheim:status
    target: real-server
    protocol: udp
    listen: [::]:2457
    flow_timeout: 5s
    options:
      name: "My Valheim Server"
      map: "Dedicated"
      version: "0.221.12"
      max_players: 10
      password: true
```

## server Frontend

The `server` frontend has no options. Every UDP datagram received on the
gameplay port is inspected before warming the target. Packets wake the target
only when they look like Steam Networking Sockets connection setup traffic for
the gameplay port; A2S/status probes and unrelated UDP packets are ignored. The
real Valheim handshake is left to the dedicated server after it starts.

## status Frontend

The `status` frontend answers these Steam query packets:

- `A2S_INFO`: returns configured server metadata.
- `A2S_PLAYER`, `A2S_RULES`, and `A2S_SERVERQUERY_GETCHALLENGE`: returns a
  challenge response.
- `A2A_PING`: returns the legacy ping response.

Options:

- `name`: Required server name shown to query clients.
- `map`: Required map/world label.
- `version`: Required version string.
- `max_players`: Required maximum player count.
- `password`: Whether to report the server as password-protected. Defaults to
  `true`.
- `vac`: Whether to report VAC as enabled. Defaults to `false`.
