# LAN TF2 Server Manager

A small Fyne GUI for monitoring and managing Team Fortress 2 servers over RCON.

## Run the app

```sh
go run .
```

- Default connection values are `0.0.0.0:27015` / `test`, but you can change them per tab. Enter the server address, RCON password, and click **Connect**.
- Check **Auto refresh** and set an interval in seconds to poll `status` automatically while connected.
- The **Server password** field under the connection form is used for the copyable connect/STV strings and for the **Set password** action.

### Tabs

- Click the **＋** icon at the end of the tab bar to add a new server tab.
- Tabs can be closed with their **×** button; closing disconnects that server.
- The last remaining tab has its close button hidden so the window never ends up empty.
- Once connected, the tab title changes from "Server N" to the server's `hostname`.
- Open tabs, addresses, passwords, and selected maps are persisted in Fyne app preferences.
  Passwords are stored in plain text, which is fine for a LAN tool but not for production use.

### Server panel

After connecting, the panel shows:

- Server address (`udp/ip`)
- Current map
- Human player count / max players
- SourceTV address, delay, and local bind
- Refresh button next to the server info title
- Copyable **Connect** string: `connect <address>; password <password>`
- Copyable **STV** string: `connect <stv_address>; password <password>`
- **Change map** dropdown with the configured map list
- **Exec config** dropdown with the configured config list
- **Set password** button to run `sv_password <password>`
- **Kick All Players** button above the player list
- Expandable player list with ID, name, Steam unique ID, and a per-player **Kick** button

### Actions

- **Change map** sends `changelevel <map>` over RCON.
- **Exec config** sends `exec <config_name>` over RCON.
- **Set password** sends `sv_password <password>` over RCON and updates the copyable connect strings.
- **Kick** on a player row sends `kickid <userid>` over RCON and refreshes the list.
- **Kick All Players** asks for confirmation in a modal dialog, then kicks every listed human player and refreshes.

### Configuration

The map and config dropdowns are populated from `config/config.toml`:

```toml
maps = [
    "cp_badlands",
    "cp_sunshine",
    "cp_process_f12",
    "cp_gullywash_f9",
    "cp_metalworks_f7",
    "koth_bagel_rc12",
    "koth_product_final",
    "cp_granary_pro_rc17a3",
    "mge_training_v8_beta4b",
]

configs = [
    "etf2l_6v6_5cp",
    "etf2l_6v6_koth",
]

# Optional: open and connect to servers automatically on startup.
# [[servers]]
# address = "0.0.0.0:27015"
# rcon_password = "test"
```

If `config/config.toml` is missing or invalid, the app falls back to the embedded default config.

When `[[servers]]` entries are present, the app opens a tab for each one and connects automatically. Servers from the config take priority over tabs saved in preferences.

## Test with Docker

For local development you can use the `spiretf/docker-spire-server` image:

```sh
# Start the server (default RCON password is "rcon")
docker compose up

# Or override the password
docker compose up -e RCON_PASSWORD=secret
```

Then connect the app to `127.0.0.1:27015` with the matching password.

## Build

```sh
go build -o server-manager .
```

## Lint

```sh
make lint
```

This runs, in order:

- `gofmt -w .`
- `go run golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@latest -fix ./...`
- `go vet ./...`
- `golangci-lint run ./...`

## Package

```sh
fyne package -release
```
