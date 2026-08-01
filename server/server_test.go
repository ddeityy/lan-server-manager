package server

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

	if info.Hostname != "test" {
		t.Errorf("hostname = %q, want %q", info.Hostname, "test")
	}
	if info.Address != "169.254.77.194:47656" {
		t.Errorf("address = %q, want %q", info.Address, "169.254.77.194:47656")
	}
	if info.Map != "cp_badlands" {
		t.Errorf("map = %q, want %q", info.Map, "cp_badlands")
	}
	if info.SourceTV.Address != "169.254.77.194:47656" {
		t.Errorf("sourcetv address = %q, want %q", info.SourceTV.Address, "169.254.77.194:47656")
	}
	if info.SourceTV.Delay != "90.0s" {
		t.Errorf("sourcetv delay = %q, want %q", info.SourceTV.Delay, "90.0s")
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
	if player.Connected != "00:06" {
		t.Errorf("connected = %q, want %q", player.Connected, "00:06")
	}
	if player.Ping != 61 {
		t.Errorf("ping = %d, want 61", player.Ping)
	}
	if player.Loss != 22 {
		t.Errorf("loss = %d, want 22", player.Loss)
	}
	if player.State != "active" {
		t.Errorf("state = %q, want %q", player.State, "active")
	}
}
