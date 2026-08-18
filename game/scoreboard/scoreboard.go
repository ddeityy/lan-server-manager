package scoreboard

import (
	"maps"
	"strconv"
	"sync"
	"time"

	"lan-server-manager/game/logparse"
)

// Snapshot is an immutable value copy of the scoreboard at one point in time.
type Snapshot struct {
	Red        []PlayerStats
	Blu        []PlayerStats
	Spec       []PlayerStats
	Unassigned []PlayerStats
	RedScore   int
	BluScore   int
	Elapsed    time.Duration
	CVars      map[string]string
}

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

// perMinute converts an absolute count to a per-minute rate based on elapsed time.
func perMinute(value int, elapsed time.Duration) float64 {
	m := elapsed.Minutes()
	if m <= 0 {
		return 0
	}
	return float64(value) / m
}

// DPM returns damage per minute since the game started.
func (p *PlayerStats) DPM(elapsed time.Duration) float64 {
	return perMinute(p.Damage, elapsed)
}

// DTM returns damage taken per minute since the game started.
func (p *PlayerStats) DTM(elapsed time.Duration) float64 {
	return perMinute(p.DamageTaken, elapsed)
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

	// cvars stores the latest echoed value for any tracked CVar.
	cvars map[string]string
}

// New creates an empty scoreboard.
func New() *Scoreboard {
	return &Scoreboard{
		players: make(map[string]*PlayerStats),
		cvars:   make(map[string]string),
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
		if evt.Data["cvar"] != "" {
			s.cvars[evt.Data["cvar"]] = evt.Data["value"]
		}

	case logparse.EventStatusSeed:
		p := s.upsertLocked(evt.Source)
		if evt.Source.Ping > 0 {
			p.Ping = evt.Source.Ping
		}

	case logparse.EventReset:
		s.resetLocked()
	}
}

// Upsert returns an existing player entry or creates one. The public entry
// point for seeding players from RCON status.
func (s *Scoreboard) Upsert(player logparse.Player) *PlayerStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.upsertLocked(player)
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
func (s *Scoreboard) upsertLocked(player logparse.Player) *PlayerStats {
	key := playerKey(player)
	if key == "" {
		return &PlayerStats{Name: player.Name}
	}
	stats, ok := s.players[key]
	if !ok {
		stats = &PlayerStats{
			SteamID: player.SteamID,
			UserID:  player.UserID,
			Name:    player.Name,
			Team:    player.Team,
		}
		s.players[key] = stats
	} else if player.Name != "" {
		stats.Name = player.Name
		stats.UserID = player.UserID
		if player.SteamID != "" && player.SteamID != "BOT" {
			stats.SteamID = player.SteamID
		}
		if player.Team != logparse.TeamUnknown {
			stats.Team = player.Team
		}
		if player.Ping > 0 {
			stats.Ping = player.Ping
		}
	}
	return stats
}

// playerKey returns a unique key for a log player. SteamID is preferred, but
// falls back to user ID for bots or missing SteamIDs so multiple bots don't
// collapse into a single entry.
func playerKey(player logparse.Player) string {
	if player.SteamID != "" && player.SteamID != "BOT" {
		return player.SteamID
	}
	if player.UserID != "" {
		return "UID:" + player.UserID
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
	s.cvars = make(map[string]string)
}

// Player returns a player entry by SteamID or composite key.
func (s *Scoreboard) Player(key string) (*PlayerStats, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	player, ok := s.players[key]
	if !ok {
		return nil, false
	}
	return player, true
}

// Remove deletes a player from the scoreboard by SteamID or composite key.
func (s *Scoreboard) Remove(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.players, key)
}

// Teams returns the current Red, Blu, and Spectator rosters as value copies.
func (s *Scoreboard) Teams() ([]PlayerStats, []PlayerStats, []PlayerStats) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var red, blu, spec []PlayerStats
	for _, player := range s.players {
		switch player.Team {
		case logparse.TeamRed:
			red = append(red, *player)
		case logparse.TeamBlu:
			blu = append(blu, *player)
		case logparse.TeamSpec:
			spec = append(spec, *player)
		}
	}
	return red, blu, spec
}

// TeamsAndUnassigned returns the Red, Blu, Spectator, and unassigned rosters.
func (s *Scoreboard) TeamsAndUnassigned() ([]PlayerStats, []PlayerStats, []PlayerStats, []PlayerStats) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var red, blu, spec, unassigned []PlayerStats
	for _, player := range s.players {
		switch player.Team {
		case logparse.TeamRed:
			red = append(red, *player)
		case logparse.TeamBlu:
			blu = append(blu, *player)
		case logparse.TeamSpec:
			spec = append(spec, *player)
		default:
			unassigned = append(unassigned, *player)
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
func (s *Scoreboard) Scores() (int, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.redScore, s.bluScore
}

// CVar returns the latest echoed value for a tracked CVar.
func (s *Scoreboard) CVar(name string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cvars[name]
}

// Snapshot returns an immutable value copy of the current scoreboard state.
func (s *Scoreboard) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	red, blu, spec, unassigned := s.teamsAndUnassignedLocked()
	cvars := maps.Clone(s.cvars)

	return Snapshot{
		Red:        red,
		Blu:        blu,
		Spec:       spec,
		Unassigned: unassigned,
		RedScore:   s.redScore,
		BluScore:   s.bluScore,
		Elapsed:    s.elapsedLocked(),
		CVars:      cvars,
	}
}

func (s *Scoreboard) teamsAndUnassignedLocked() ([]PlayerStats, []PlayerStats, []PlayerStats, []PlayerStats) {
	var red, blu, spec, unassigned []PlayerStats
	for _, player := range s.players {
		switch player.Team {
		case logparse.TeamRed:
			red = append(red, *player)
		case logparse.TeamBlu:
			blu = append(blu, *player)
		case logparse.TeamSpec:
			spec = append(spec, *player)
		default:
			unassigned = append(unassigned, *player)
		}
	}
	return red, blu, spec, unassigned
}

func (s *Scoreboard) elapsedLocked() time.Duration {
	if s.gameStart.IsZero() {
		return 0
	}
	return time.Since(s.gameStart)
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
