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
	if info.HumanPlayers != 0 {
		t.Errorf("human players = %d, want 0", info.HumanPlayers)
	}
	if info.MaxPlayers != 25 {
		t.Errorf("max players = %d, want 25", info.MaxPlayers)
	}
	if len(info.Players) != 0 {
		t.Errorf("players = %d, want 0 (BOT should be excluded)", len(info.Players))
	}
}
