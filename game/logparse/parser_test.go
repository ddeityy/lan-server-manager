package logparse

import (
	"testing"
	"time"
)

func TestParseChat(t *testing.T) {
	line := `L 08/17/2026 - 21:43:59: "Deity.<3><[U:1:115754284]><Red>" say "etst"`
	evt, ok := Parse(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if evt.Type != EventChat {
		t.Errorf("type = %d, want EventChat", evt.Type)
	}
	if evt.Source.Name != "Deity." {
		t.Errorf("name = %q, want Deity.", evt.Source.Name)
	}
	if evt.Source.UserID != "3" {
		t.Errorf("userid = %q, want 3", evt.Source.UserID)
	}
	if evt.Source.SteamID != "[U:1:115754284]" {
		t.Errorf("steamid = %q, want [U:1:115754284]", evt.Source.SteamID)
	}
	if evt.Source.Team != TeamRed {
		t.Errorf("team = %d, want TeamRed", evt.Source.Team)
	}
	if evt.Data["message"] != "etst" {
		t.Errorf("message = %q, want etst", evt.Data["message"])
	}
}

func TestParseJoinedTeam(t *testing.T) {
	line := `L 08/17/2026 - 21:44:00: "Deity.<3><[U:1:115754284]><>" joined team "Blue"`
	evt, ok := Parse(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if evt.Type != EventJoinedTeam {
		t.Errorf("type = %d, want EventJoinedTeam", evt.Type)
	}
	if evt.Source.Team != TeamBlu {
		t.Errorf("team = %d, want TeamBlu", evt.Source.Team)
	}
}

func TestParseKilled(t *testing.T) {
	line := `L 08/17/2026 - 21:44:01: "Deity.<3><[U:1:115754284]><Red>" killed "Bob.<4><[U:1:2]><Blue>" with "scattergun" (attacker_position "1 2 3") (victim_position "4 5 6")`
	evt, ok := Parse(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if evt.Type != EventKilled {
		t.Errorf("type = %d, want EventKilled", evt.Type)
	}
	if evt.Target.Name != "Bob." {
		t.Errorf("victim = %q, want Bob.", evt.Target.Name)
	}
	if evt.Target.Team != TeamBlu {
		t.Errorf("victim team = %d, want TeamBlu", evt.Target.Team)
	}
	if evt.Data["weapon"] != "scattergun" {
		t.Errorf("weapon = %q, want scattergun", evt.Data["weapon"])
	}
}

func TestParseDamage(t *testing.T) {
	line := `L 08/17/2026 - 21:44:02: "Deity.<3><[U:1:115754284]><Red>" triggered "damage" against "Bob.<4><[U:1:2]><Blue>" (damage "35") (weapon "scattergun")`
	evt, ok := Parse(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if evt.Type != EventDamage {
		t.Errorf("type = %d, want EventDamage", evt.Type)
	}
	if evt.Data["damage"] != "35" {
		t.Errorf("damage = %q, want 35", evt.Data["damage"])
	}
	if evt.Data["weapon"] != "scattergun" {
		t.Errorf("weapon = %q, want scattergun", evt.Data["weapon"])
	}
}

func TestParsePointCaptured(t *testing.T) {
	line := `L 08/17/2026 - 21:44:03: Team "Blue" triggered "pointcaptured" (cp "2") (cpname "Middle") (numcappers "2")`
	evt, ok := Parse(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if evt.Type != EventPointCaptured {
		t.Errorf("type = %d, want EventPointCaptured", evt.Type)
	}
	if evt.Data["team"] != "Blue" {
		t.Errorf("team = %q, want Blue", evt.Data["team"])
	}
}

func TestParseRoundStart(t *testing.T) {
	line := `L 08/17/2026 - 21:44:04: World triggered "Round_Start"`
	evt, ok := Parse(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if evt.Type != EventRoundStart {
		t.Errorf("type = %d, want EventRoundStart", evt.Type)
	}
}

func TestParseCVar(t *testing.T) {
	line := `L 08/17/2026 - 21:44:05: "mp_timelimit" = "111" ( def. "0" )`
	evt, ok := Parse(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if evt.Type != EventCVar {
		t.Errorf("type = %d, want EventCVar", evt.Type)
	}
	if evt.Data["cvar"] != "mp_timelimit" {
		t.Errorf("cvar = %q, want mp_timelimit", evt.Data["cvar"])
	}
	if evt.Data["value"] != "111" {
		t.Errorf("value = %q, want 111", evt.Data["value"])
	}
}

func TestParseHealed(t *testing.T) {
	line := `L 08/17/2026 - 21:44:06: "Medic.<5><[U:1:9]><Red>" triggered "healed" against "Soldier.<4><[U:1:8]><Red>" (healing "100")`
	evt, ok := Parse(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if evt.Type != EventHealed {
		t.Errorf("type = %d, want EventHealed", evt.Type)
	}
	if evt.Source.Name != "Medic." {
		t.Errorf("source name = %q, want Medic.", evt.Source.Name)
	}
	if evt.Target.Name != "Soldier." {
		t.Errorf("target name = %q, want Soldier.", evt.Target.Name)
	}
	if evt.Data["healing"] != "100" {
		t.Errorf("healing = %q, want 100", evt.Data["healing"])
	}
}

func TestParseDisconnected(t *testing.T) {
	line := `L 08/18/2026 - 01:14:55: "SomeDude<11><BOT><Blue>" disconnected (reason "Kicked from server")`
	evt, ok := Parse(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if evt.Type != EventDisconnected {
		t.Errorf("type = %d, want EventDisconnected", evt.Type)
	}
	if evt.Source.Name != "SomeDude" {
		t.Errorf("name = %q, want SomeDude", evt.Source.Name)
	}
	if evt.Source.UserID != "11" {
		t.Errorf("userid = %q, want 11", evt.Source.UserID)
	}
	if evt.Source.SteamID != "BOT" {
		t.Errorf("steamid = %q, want BOT", evt.Source.SteamID)
	}
	if evt.Data["reason"] != "Kicked from server" {
		t.Errorf("reason = %q, want Kicked from server", evt.Data["reason"])
	}
}

func TestParseServerCVar(t *testing.T) {
	line := `L 08/18/2026 - 11:20:46: server_cvar: "mp_winlimit" "4"`
	evt, ok := Parse(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if evt.Type != EventCVar {
		t.Errorf("type = %d, want EventCVar", evt.Type)
	}
	if evt.Data["cvar"] != "mp_winlimit" {
		t.Errorf("cvar = %q, want mp_winlimit", evt.Data["cvar"])
	}
	if evt.Data["value"] != "4" {
		t.Errorf("value = %q, want 4", evt.Data["value"])
	}
}

func TestParseMapChange(t *testing.T) {
	line := `-------- Mapchange to cp_badlands --------`
	evt, ok := Parse(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	if evt.Type != EventMapChange {
		t.Errorf("type = %d, want EventMapChange", evt.Type)
	}
	if evt.Data["map"] != "cp_badlands" {
		t.Errorf("map = %q, want cp_badlands", evt.Data["map"])
	}
}

func TestParseTimestamp(t *testing.T) {
	line := `L 08/17/2026 - 21:43:59: "Deity.<3><[U:1:115754284]><Red>" say "etst"`
	evt, ok := Parse(line)
	if !ok {
		t.Fatalf("expected parse success")
	}
	want := time.Date(2026, time.August, 17, 21, 43, 59, 0, time.Local)
	if !evt.Timestamp.Equal(want) {
		t.Errorf("timestamp = %v, want %v", evt.Timestamp, want)
	}
}
