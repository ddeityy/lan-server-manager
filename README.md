# LAN TF2 Server Manager

A small Fyne GUI for monitoring and managing Team Fortress 2 servers over RCON on a LAN.

## Run

```sh
go run .
```

Or build a binary:

```sh
go build -o server-manager .
```

## Configuration

The map list, config list, and preloaded servers come from `config/config.toml`. If the file is missing or a list is empty, the app falls back to the embedded defaults.

To override the defaults, create or edit `config/config.toml`:

```toml
maps = [
    "cp_badlands",
    "cp_process_f12",
]

configs = [
    "etf2l_6v6_5cp",
]

[[servers]]
address = "127.0.0.1:27015"
rcon_password = "dsadsadsadsadsada"
password = ""
```

When `[[servers]]` entries are present, the app opens a tab for each one, connects automatically, and optionally sets the server password.

## MoscowLAN setup

The default config points at the servers in `msk-lan-compose.yml`:

| Address           | Server              |
|-------------------|---------------------|
| `127.0.0.1:27015` | MoscowLAN Server #1 |
| `127.0.0.1:27025` | MoscowLAN Server #2 |
| `127.0.0.1:27035` | MoscowLAN Server #3 |
| `127.0.0.1:27045` | MoscowLAN MGE Server |

Start them with:

```sh
docker compose -f msk-lan-compose.yml up
```
