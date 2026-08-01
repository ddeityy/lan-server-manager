# LAN TF2 Server Manager

A small Fyne GUI for monitoring and managing Team Fortress 2 servers over RCON.

## Run the app

```sh
go run .
```

- Default connection values are `0.0.0.0:27015` / `test`, but you can change them per tab. Enter the server address, RCON password, and click **Connect**.
- Check **Auto refresh** and set an interval in seconds to poll `status` automatically while connected.

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
- Refresh button above the server info
- **Change Level** dropdown with common competitive maps
- **Kick All Players** button above the player list
- Expandable player list with ID, name, Steam unique ID, and a per-player **Kick** button

### Kick actions

- **Kick** on a player row sends `kickid <userid>` over RCON and refreshes the list.
- **Kick All Players** asks for confirmation in a modal dialog, then kicks every listed human player and refreshes.

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

## Package

```sh
fyne package -release
```
