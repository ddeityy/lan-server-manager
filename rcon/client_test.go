package rcon

import (
	"os"
	"testing"
)

func TestParseStatus(t *testing.T) {
	data, err := os.ReadFile("testdata/example_rcon_status_output.txt")
	if err != nil {
		t.Fatalf("read example file: %v", err)
	}

	info, err := ParseStatus(string(data))
	if err != nil {
		t.Fatalf("parse status: %v", err)
	}

	if info.Hostname != "MoscowLAN Server №1" {
		t.Errorf("hostname = %q, want %q", info.Hostname, "MoscowLAN Server №1")
	}
	if info.Address.SDR != "169.254.12.131:27776" {
		t.Errorf("address sdr = %q, want %q", info.Address.SDR, "169.254.12.131:27776")
	}
	if info.Address.Local != "0.0.0.0:27015" {
		t.Errorf("address local = %q, want %q", info.Address.Local, "0.0.0.0:27015")
	}
	if info.Map != "cp_badlands" {
		t.Errorf("map = %q, want %q", info.Map, "cp_badlands")
	}
	if info.SourceTV.Address != "169.254.12.131:27776" {
		t.Errorf("sourcetv address = %q, want %q", info.SourceTV.Address, "169.254.12.131:27776")
	}
	if info.SourceTV.Delay != "30.0s" {
		t.Errorf("sourcetv delay = %q, want %q", info.SourceTV.Delay, "30.0s")
	}
	if info.SourceTV.Local != "0.0.0.0:27020" {
		t.Errorf("sourcetv local = %q, want %q", info.SourceTV.Local, "0.0.0.0:27020")
	}
	if info.HumanPlayers != 1 {
		t.Errorf("human players = %d, want 1", info.HumanPlayers)
	}
	if info.MaxPlayers != 25 {
		t.Errorf("max players = %d, want 25", info.MaxPlayers)
	}
	if len(info.Players) != 1 {
		t.Fatalf("players = %d, want 1", len(info.Players))
	}

	player := info.Players[0]
	if player.UserID != 3 {
		t.Errorf("userid = %d, want 3", player.UserID)
	}
	if player.Name != "Deity. #NextYear" {
		t.Errorf("name = %q, want %q", player.Name, "Deity. #NextYear")
	}
	if player.UniqueID != "[U:1:115754284]" {
		t.Errorf("uniqueid = %q, want %q", player.UniqueID, "[U:1:115754284]")
	}
	if player.Connected != "00:37" {
		t.Errorf("connected = %q, want %q", player.Connected, "00:37")
	}
	if player.Ping != 74 {
		t.Errorf("ping = %d, want 74", player.Ping)
	}
	if player.Loss != 0 {
		t.Errorf("loss = %d, want 0", player.Loss)
	}
	if player.State != "active" {
		t.Errorf("state = %q, want %q", player.State, "active")
	}
	if player.Address != "169.254.254.59:28360" {
		t.Errorf("address = %q, want %q", player.Address, "169.254.254.59:28360")
	}
}
