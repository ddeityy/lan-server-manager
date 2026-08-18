# TF2 Server Manager for LANs

A small Fyne GUI for monitoring and managing Team Fortress 2 servers over RCON / SSH in a LAN environment.

Battle tested on [MoscowLAN 2026](https://liquipedia.net/teamfortress/Moscow_Lan/2026) with 5 servers running concurrent matches for 2 days.

## Run or build

```sh
go run .
```

```sh
go build -o server-manager .
```

For dependencies and packaging see [fyne documentation](https://docs.fyne.io/started/quick)

## Configuration

The map list, config list, and preloaded servers come from `config/config.toml`. If the file is missing or a list is empty, the app falls back to the embedded defaults in the [default config file](https://github.com/ddeityy/moscow-lan-server-manager/blob/main/config/config.toml).

To override the defaults, create or edit `config/config.toml`:

```toml
maps = [
    "cp_badlands",
    "cp_process_f12",
    "your_custom_map"
]

configs = [
    "etf2l_6v6_5cp",
    "your_custom_config"
]

[[servers]]
address = "127.0.0.1:27015"
rcon_password = "password"
password = ""
container_name = "LanServer1"
# if left empty will default to local containers
ssh_host = ""
ssh_user = ""
ssh_password = ""
ssh_key_path = ""
```

When `[[servers]]` entries are present, the app opens a tab for each one, connects automatically, and optionally sets the server password.

## Acknowledgments

Log parsing logic and event identifiers are based on [logstf](https://github.com/leighmacdonald/logstf) by [leighmacdonald](https://github.com/leighmacdonald).
