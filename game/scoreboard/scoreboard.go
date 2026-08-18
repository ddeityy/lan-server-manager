package scoreboard

import (
	"strconv"
	"sync"
	"time"

	"lan-server-manager/game/logparse"
)

// PlayerStats holds the scoreboard numbers for a single player.
type PlayerStats struct {
	SteamID     string
	UserID      string
	Name        string
	Team        logparse.Team
	Class       logparse.PlayerClass
	Ping        int
	Kills       int
	Assists     int
	Deaths      int
	Damage      int
	DamageTaken int
	Caps        int
	Heals       int
}

// DPM returns damage per minute since the game started.
func (p *PlayerStats) DPM(elapsed time.Duration) float64 {
	m := elapsed.Minutes()
	if m <= 0 {
		return 0
	}
	return float64(p.Damage) / m
}

// DTM returns damage taken per minute since the game started.
func (p *PlayerStats) DTM(elapsed time.Duration) float64 {
	m := elapsed.Minutes()
	if m <= 0 {
		return 0
	}
	return float64(p.DamageTaken) / m
}

// KAD returns (kills + assists) / deaths.
func (p *PlayerStats) KAD() float64 {
	if p.Deaths == 0 {
		return float64(p.Kills + p.Assists)
	}
	return float64(p.Kills+p.Assists) / float64(p.Deaths)
}

// KD returns kills / deaths.
func (p *PlayerStats) KD() float64 {
	if p.Deaths == 0 {
		return float64(p.Kills)
	}
	return float64(p.Kills) / float64(p.Deaths)
}

// Scoreboard holds the live state of a single game.
type Scoreboard struct {
	mu        sync.RWMutex
	players   map[string]*PlayerStats
	gameStart time.Time

	redScore int
	bluScore int

	timelimit     string
	winlimit      string
	windifference string
}

// New creates an empty scoreboard.
func New() *Scoreboard {
	return &Scoreboard{
		players: make(map[string]*PlayerStats),
	}
}

// Apply updates the scoreboard from a parsed log event.
func (s *Scoreboard) Apply(evt logparse.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch evt.Type {
	case logparse.EventMapChange:
		s.resetLocked()

	case logparse.EventRoundStart:
		if s.gameStart.IsZero() {
			s.gameStart = evt.Timestamp
		}

	case logparse.EventJoinedTeam:
		p := s.upsertLocked(evt.Source)
		p.Team = evt.Source.Team

	case logparse.EventChangeClass, logparse.EventSpawned:
		p := s.upsertLocked(evt.Source)
		p.Class = logparse.ClassFromString(evt.Data["class"])

	case logparse.EventKilled:
		if evt.Source.Team == logparse.TeamSpec || evt.Target.Team == logparse.TeamSpec {
			return
		}
		attacker := s.upsertLocked(evt.Source)
		attacker.Kills++
		victim := s.upsertLocked(evt.Target)
		victim.Deaths++

	case logparse.EventKillAssist:
		if evt.Source.Team == logparse.TeamSpec {
			return
		}
		assister := s.upsertLocked(evt.Source)
		assister.Assists++

	case logparse.EventDamage:
		dmg, _ := strconv.Atoi(evt.Data["damage"])
		if dmg <= 0 {
			return
		}
		if evt.Source.Team == logparse.TeamSpec || evt.Target.Team == logparse.TeamSpec {
			return
		}
		attacker := s.upsertLocked(evt.Source)
		attacker.Damage += dmg
		victim := s.upsertLocked(evt.Target)
		victim.DamageTaken += dmg

	case logparse.EventHealed:
		heal, _ := strconv.Atoi(evt.Data["healing"])
		if heal <= 0 {
			return
		}
		p := s.upsertLocked(evt.Source)
		p.Heals += heal

	case logparse.EventDisconnected:
		key := playerKey(evt.Source)
		if key != "" {
			delete(s.players, key)
		}

	case logparse.EventPointCaptured:
		for _, capper := range logparse.FindPlayers(evt.Data["cappers_raw"]) {
			if capper.Team == logparse.TeamSpec {
				continue
			}
			p := s.upsertLocked(capper)
			p.Caps++
		}

	case logparse.EventTeamScore, logparse.EventTeamFinalScore:
		score, _ := strconv.Atoi(evt.Data["score"])
		switch evt.Data["team"] {
		case "Red":
			s.redScore = score
		case "Blue":
			s.bluScore = score
		}

	case logparse.EventCVar:
		switch evt.Data["cvar"] {
		case cvarTimelimit:
			s.timelimit = evt.Data["value"]
		case cvarWinlimit:
			s.winlimit = evt.Data["value"]
		case cvarWindifference:
			s.windifference = evt.Data["value"]
		}
	}
}

// Upsert returns an existing player entry or creates one. The public entry
// point for seeding players from RCON status.
func (s *Scoreboard) Upsert(pl logparse.Player) *PlayerStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.upsertLocked(pl)
}

// PlayerKey returns the internal key for a user ID / SteamID pair. Useful when
// callers already know one or both identifiers and need to address a player.
func PlayerKey(userID, steamID string) string {
	return playerKey(logparse.Player{UserID: userID, SteamID: steamID})
}

// SetPing updates the ping for an existing tracked player. The caller should pass
// the key returned by PlayerKey. Unknown keys are ignored.
func (s *Scoreboard) SetPing(key string, ping int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.players[key]; ok && ping > 0 {
		p.Ping = ping
	}
}

// upsertLocked returns an existing player entry or creates one. The caller must
// hold the write lock.
func (s *Scoreboard) upsertLocked(pl logparse.Player) *PlayerStats {
	key := playerKey(pl)
	if key == "" {
		return &PlayerStats{Name: pl.Name}
	}
	p, ok := s.players[key]
	if !ok {
		p = &PlayerStats{
			SteamID: pl.SteamID,
			UserID:  pl.UserID,
			Name:    pl.Name,
			Team:    pl.Team,
		}
		s.players[key] = p
	} else if pl.Name != "" {
		p.Name = pl.Name
		p.UserID = pl.UserID
		if pl.SteamID != "" && pl.SteamID != "BOT" {
			p.SteamID = pl.SteamID
		}
		if pl.Team != logparse.TeamUnknown {
			p.Team = pl.Team
		}
		if pl.Ping > 0 {
			p.Ping = pl.Ping
		}
	}
	return p
}

// playerKey returns a unique key for a log player. SteamID is preferred, but
// falls back to user ID for bots or missing SteamIDs so multiple bots don't
// collapse into a single entry.
func playerKey(pl logparse.Player) string {
	if pl.SteamID != "" && pl.SteamID != "BOT" {
		return pl.SteamID
	}
	if pl.UserID != "" {
		return "UID:" + pl.UserID
	}
	return ""
}

// Reset clears all players and game state.
func (s *Scoreboard) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resetLocked()
}

func (s *Scoreboard) resetLocked() {
	s.players = make(map[string]*PlayerStats)
	s.gameStart = time.Time{}
	s.redScore = 0
	s.bluScore = 0
}

// Player returns a player entry by SteamID or composite key.
func (s *Scoreboard) Player(key string) (*PlayerStats, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.players[key]
	if !ok {
		return nil, false
	}
	return p, true
}

// Remove deletes a player from the scoreboard by SteamID or composite key.
func (s *Scoreboard) Remove(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.players, key)
}

// Teams returns the current Red, Blu, and Spectator rosters as value copies.
func (s *Scoreboard) Teams() (red, blu, spec []PlayerStats) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.players {
		switch p.Team {
		case logparse.TeamRed:
			red = append(red, *p)
		case logparse.TeamBlu:
			blu = append(blu, *p)
		case logparse.TeamSpec:
			spec = append(spec, *p)
		}
	}
	return red, blu, spec
}

// TeamsAndUnassigned returns the Red, Blu, Spectator, and unassigned rosters.
func (s *Scoreboard) TeamsAndUnassigned() (red, blu, spec, unassigned []PlayerStats) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.players {
		switch p.Team {
		case logparse.TeamRed:
			red = append(red, *p)
		case logparse.TeamBlu:
			blu = append(blu, *p)
		case logparse.TeamSpec:
			spec = append(spec, *p)
		default:
			unassigned = append(unassigned, *p)
		}
	}
	return red, blu, spec, unassigned
}

// TimeSinceStart returns how long the current round/map has been running.
func (s *Scoreboard) TimeSinceStart() time.Duration {
	s.mu.RLock()
	start := s.gameStart
	s.mu.RUnlock()
	if start.IsZero() {
		return 0
	}
	return time.Since(start)
}

// GameStart returns the timestamp of the current round start, if known.
func (s *Scoreboard) GameStart() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gameStart
}

// SetGameStart forces the game-start timestamp. Useful for seeding from RCON.
func (s *Scoreboard) SetGameStart(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gameStart = t
}

// Scores returns the current Red and Blu team scores.
func (s *Scoreboard) Scores() (red, blu int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.redScore, s.bluScore
}

const (
	cvarTimelimit     = "mp_timelimit"
	cvarWinlimit      = "mp_winlimit"
	cvarWindifference = "mp_windifference"
)

// CVar returns the latest echoed values for the tracked cvars.
func (s *Scoreboard) CVar(name string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	switch name {
	case cvarTimelimit:
		return s.timelimit
	case cvarWinlimit:
		return s.winlimit
	case cvarWindifference:
		return s.windifference
	}
	return ""
}

// AllPlayers returns a copy of every tracked player. Spectators are included
// but carry no stats.
func (s *Scoreboard) AllPlayers() []*PlayerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*PlayerStats, 0, len(s.players))
	for _, p := range s.players {
		out = append(out, p)
	}
	return out
}
