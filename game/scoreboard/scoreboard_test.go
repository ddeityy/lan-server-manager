package scoreboard

import (
	"testing"
	"time"

	"lan-server-manager/game/logparse"
)

func TestApplyDamageAndKill(t *testing.T) {
	sb := New()
	sb.SetGameStart(time.Now().Add(-time.Minute))

	sb.Apply(logparse.Event{
		Type:      logparse.EventDamage,
		Timestamp: time.Now(),
		Source:    logparse.Player{Name: "A", SteamID: "1", Team: logparse.TeamRed},
		Target:    logparse.Player{Name: "B", SteamID: "2", Team: logparse.TeamBlu},
		Data:      map[string]string{"damage": "100"},
	})

	if p, _ := sb.Player("1"); p.Damage != 100 {
		t.Errorf("attacker damage = %d, want 100", p.Damage)
	}
	if p, _ := sb.Player("2"); p.DamageTaken != 100 {
		t.Errorf("victim damage taken = %d, want 100", p.DamageTaken)
	}

	sb.Apply(logparse.Event{
		Type:      logparse.EventKilled,
		Timestamp: time.Now(),
		Source:    logparse.Player{Name: "A", SteamID: "1", Team: logparse.TeamRed},
		Target:    logparse.Player{Name: "B", SteamID: "2", Team: logparse.TeamBlu},
		Data:      map[string]string{"weapon": "scattergun"},
	})

	if p, _ := sb.Player("1"); p.Kills != 1 {
		t.Errorf("attacker kills = %d, want 1", p.Kills)
	}
	if p, _ := sb.Player("2"); p.Deaths != 1 {
		t.Errorf("victim deaths = %d, want 1", p.Deaths)
	}
}

func TestSpectatorsNoStats(t *testing.T) {
	sb := New()
	sb.Apply(logparse.Event{
		Type:      logparse.EventKilled,
		Timestamp: time.Now(),
		Source:    logparse.Player{Name: "SpecPlayer", SteamID: "9", Team: logparse.TeamSpec},
		Target:    logparse.Player{Name: "RedPlayer", SteamID: "1", Team: logparse.TeamRed},
		Data:      map[string]string{},
	})

	if _, ok := sb.Player("9"); ok {
		t.Errorf("spectator should not be tracked")
	}
	if _, ok := sb.Player("1"); ok {
		t.Errorf("kills by spectators should not affect other players")
	}
}

func TestMapChangeResets(t *testing.T) {
	sb := New()
	sb.SetGameStart(time.Now().Add(-time.Minute))
	sb.Apply(logparse.Event{
		Type:      logparse.EventKilled,
		Timestamp: time.Now(),
		Source:    logparse.Player{Name: "A", SteamID: "1", Team: logparse.TeamRed},
		Target:    logparse.Player{Name: "B", SteamID: "2", Team: logparse.TeamBlu},
		Data:      map[string]string{},
	})

	sb.Apply(logparse.Event{Type: logparse.EventMapChange, Data: map[string]string{"map": "cp_process_f12"}})

	if _, ok := sb.Player("1"); ok {
		t.Errorf("expected scoreboard to be empty after map change")
	}
	if sb.GameStart() != (time.Time{}) {
		t.Errorf("expected game start to be reset")
	}
}

func TestCVarStorage(t *testing.T) {
	sb := New()
	sb.Apply(logparse.Event{
		Type:      logparse.EventCVar,
		Timestamp: time.Now(),
		Data:      map[string]string{"cvar": "mp_timelimit", "value": "30"},
	})

	if got := sb.CVar("mp_timelimit"); got != "30" {
		t.Errorf("timelimit = %q, want 30", got)
	}
}

func TestPointCapture(t *testing.T) {
	sb := New()
	sb.Apply(logparse.Event{
		Type:      logparse.EventPointCaptured,
		Timestamp: time.Now(),
		Data: map[string]string{
			"team":        "Blue",
			"cappers_raw": `"A<1><STEAM:1><Blue>" "B<2><STEAM:2><Blue>"`,
		},
	})

	if p, _ := sb.Player("STEAM:1"); p.Caps != 1 {
		t.Errorf("capper A caps = %d, want 1", p.Caps)
	}
	if p, _ := sb.Player("STEAM:2"); p.Caps != 1 {
		t.Errorf("capper B caps = %d, want 1", p.Caps)
	}
}

func TestDisconnectedRemovesPlayer(t *testing.T) {
	sb := New()
	sb.Apply(logparse.Event{
		Type:      logparse.EventJoinedTeam,
		Timestamp: time.Now(),
		Source:    logparse.Player{Name: "SomeDude", UserID: "11", Team: logparse.TeamBlu},
	})
	if _, ok := sb.Player("UID:11"); !ok {
		t.Fatalf("expected player to be tracked before disconnect")
	}

	sb.Apply(logparse.Event{
		Type:      logparse.EventDisconnected,
		Timestamp: time.Now(),
		Source:    logparse.Player{Name: "SomeDude", UserID: "11", Team: logparse.TeamBlu},
		Data:      map[string]string{"reason": "Kicked from server"},
	})

	if _, ok := sb.Player("UID:11"); ok {
		t.Errorf("expected player to be removed after disconnect")
	}
}

func TestHealed(t *testing.T) {
	sb := New()
	sb.Apply(logparse.Event{
		Type:      logparse.EventHealed,
		Timestamp: time.Now(),
		Source:    logparse.Player{Name: "Medic", SteamID: "1", Team: logparse.TeamRed},
		Target:    logparse.Player{Name: "Soldier", SteamID: "2", Team: logparse.TeamRed},
		Data:      map[string]string{"healing": "75"},
	})

	if p, _ := sb.Player("1"); p.Heals != 75 {
		t.Errorf("heals = %d, want 75", p.Heals)
	}
}

func TestRoundStartSetsGameStart(t *testing.T) {
	sb := New()
	start := time.Date(2026, time.August, 17, 12, 0, 0, 0, time.Local)
	sb.Apply(logparse.Event{Type: logparse.EventRoundStart, Timestamp: start})

	if got := sb.GameStart(); !got.Equal(start) {
		t.Errorf("game start = %v, want %v", got, start)
	}
}
