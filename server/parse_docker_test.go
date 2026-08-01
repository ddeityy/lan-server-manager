package server

import "testing"

func TestParseDockerStatusOutput(t *testing.T) {
	status := `hostname: "MoscowLAN Server
version : 10828683/24 10828683 secure
udp/ip  : ?.?.?.?:?  (public IP from Steam: 91.77.160.217)
steamid : [A:1:3315589142:50767] (90290038467715094)
account : not logged in  (No account specified)
map     : cp_badlands at: 0 x, 0 y, 0 z
tags    : cp
sourcetv:  ?.?.?.?:?, delay 30.0s  (local: 0.0.0.0:27020)
players : 0 humans, 1 bots (25 max)
edicts  : 416 used of 2048 max
# userid name                uniqueid            connected ping loss state  adr
#      2 "Source TV"         BOT                                     active
`

	info, err := ParseStatus(status)
	if err != nil {
		t.Fatalf("parse status: %v", err)
	}

	if info.Hostname != "MoscowLAN Server" {
		t.Errorf("hostname = %q, want %q", info.Hostname, "MoscowLAN Server")
	}
	if info.Address.SDR != "?.?.?.?:?" {
		t.Errorf("sdr = %q, want %q", info.Address.SDR, "?.?.?.?:?")
	}
	if info.Address.Local != "" {
		t.Errorf("local = %q, want empty", info.Address.Local)
	}
	if info.Address.Public != "91.77.160.217" {
		t.Errorf("public = %q, want %q", info.Address.Public, "91.77.160.217")
	}
	if info.Map != "cp_badlands" {
		t.Errorf("map = %q, want %q", info.Map, "cp_badlands")
	}
	if info.SourceTV.Address != "?.?.?.?:?" {
		t.Errorf("sourcetv address = %q, want %q", info.SourceTV.Address, "?.?.?.?:?")
	}
	if info.SourceTV.Delay != "30.0s" {
		t.Errorf("sourcetv delay = %q, want %q", info.SourceTV.Delay, "30.0s")
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
		t.Errorf("players = %d, want 0", len(info.Players))
	}
}

func TestRefreshFillsConfiguredAddress(t *testing.T) {
	s := NewServer("127.0.0.1:27015", "test")
	/* Simulate a parsed status with unusable addresses. */
	s.lastInfo = ServerInfo{
		Address:  Address{SDR: "?.?.?.?:?"},
		SourceTV: SourceTV{Address: "?.?.?.?:?", Local: "0.0.0.0:27020"},
	}

	/*
		We can't call Refresh without a real server, but we can verify the
		helpers behave correctly with ConfiguredAddress set manually.
	*/
	s.lastInfo.ConfiguredAddress = s.address

	if got := s.lastInfo.GameConnectAddress(); got != "127.0.0.1:27015" {
		t.Errorf("GameConnectAddress() = %q, want %q", got, "127.0.0.1:27015")
	}
	if got := s.lastInfo.STVConnectAddress(); got != "127.0.0.1:27020" {
		t.Errorf("STVConnectAddress() = %q, want %q", got, "127.0.0.1:27020")
	}
}
