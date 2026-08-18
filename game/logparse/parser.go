package logparse

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// EventType classifies a single Source engine log line.
type EventType int

const (
	EventUnknown EventType = iota
	EventChat
	EventChatTeam
	EventJoinedTeam
	EventChangeClass
	EventSpawned
	EventSuicide
	EventKilled
	EventKillAssist
	EventDamage
	EventPointCaptured
	EventCaptureBlocked
	EventRoundStart
	EventRoundWin
	EventRoundOvertime
	EventRoundLength
	EventTeamScore
	EventTeamFinalScore
	EventGameOver
	EventGamePaused
	EventGameUnpaused
	EventConnected
	EventDisconnected
	EventEntered
	EventValidated
	EventCVar
	EventMapChange
	EventHealed
	EventStatusSeed
	EventReset
)

// Team is one of the three in-game teams.
type Team int

const (
	TeamUnknown Team = iota
	TeamSpec
	TeamRed
	TeamBlu
)

// PlayerClass enumerates the playable classes.
type PlayerClass int

const (
	ClassUnknown PlayerClass = iota
	ClassScout
	ClassSoldier
	ClassPyro
	ClassDemoman
	ClassHeavy
	ClassEngineer
	ClassMedic
	ClassSniper
	ClassSpy
	ClassSpectator
)

// Player identifies a player inside a log event or a player seeded from RCON status.
type Player struct {
	Name    string
	UserID  string
	SteamID string
	Team    Team
	Ping    int
}

// Event is the result of parsing a single log line.
type Event struct {
	Type      EventType
	Timestamp time.Time
	Source    Player
	Target    Player
	Data      map[string]string
}

func parseTeam(s string) Team {
	switch strings.ToLower(s) {
	case "red":
		return TeamRed
	case "blue", "blu":
		return TeamBlu
	case "spectator", "spec", "unassigned":
		return TeamSpec
	default:
		return TeamUnknown
	}
}

func parseClass(s string) PlayerClass {
	switch strings.ToLower(s) {
	case "scout":
		return ClassScout
	case "soldier":
		return ClassSoldier
	case "pyro":
		return ClassPyro
	case "demoman":
		return ClassDemoman
	case "heavyweapons", "heavy":
		return ClassHeavy
	case "engineer":
		return ClassEngineer
	case "medic":
		return ClassMedic
	case "sniper":
		return ClassSniper
	case "spy":
		return ClassSpy
	default:
		return ClassUnknown
	}
}

func parseTime(date, t string) time.Time {
	timestamp, err := time.ParseInLocation("01/02/2006 - 15:04:05", date+" - "+t, time.Local)
	if err != nil {
		return time.Time{}
	}
	return timestamp
}

var (
	rxDatePrefix = regexp.MustCompile(`^L (\d{2}/\d{2}/\d{4}) - (\d{2}:\d{2}:\d{2}):\s+`)
	rxMapChange  = regexp.MustCompile(`^-------- Mapchange to (\S+) --------$`)

	rxPlayerPrefix = regexp.MustCompile(`^"([^<]+)<(\d+)><([^>]+)><([^>]*)>"\s*`)
	rxPlayerBlock  = regexp.MustCompile(`"([^<]+)<(\d+)><([^>]+)><([^>]*)>"`)

	rxChat      = regexp.MustCompile(`^say "(.*)"$`)
	rxChatTeam  = regexp.MustCompile(`^say_team "(.*)"$`)
	rxJoined    = regexp.MustCompile(`^joined team "(Red|Blue|Spectator)"$`)
	rxClass     = regexp.MustCompile(`^changed role to "(.+?)"$`)
	rxSpawned   = regexp.MustCompile(`^spawned as "(\S+)"$`)
	rxSuicide   = regexp.MustCompile(`^committed suicide with "world" \(attacker_position "(.+?)"\)$`)
	rxKilled    = regexp.MustCompile(`^killed "([^<]+)<(\d+)><([^>]+)><([^>]*)>" with "(.+?)" \(attacker_position "(.+?)"\) \(victim_position "(.+?)"\)$`)
	rxAssist    = regexp.MustCompile(`^triggered "kill assist" against "([^<]+)<(\d+)><([^>]+)><([^>]*)>" \(assister_position "(.+?)"\) \(attacker_position "(.+?)"\) \(victim_position "(.+?)"\)$`)
	rxDamageNew = regexp.MustCompile(`^triggered "damage" against "([^<]+)<(\d+)><([^>]+)><([^>]*)>"\s+(.*)$`)
	rxDamageOld = regexp.MustCompile(`^triggered "damage" \(damage "(\d+)"\)$`)

	rxCaptureBlocked = regexp.MustCompile(`^triggered "captureblocked" \(cp "(\d+)"\) \(cpname "(.+?)"\) \(position "(.+?)"\)$`)
	rxHealed         = regexp.MustCompile(`^triggered "healed" against "([^<]+)<(\d+)><([^>]+)><([^>]*)>"\s+(.*)$`)

	rxPointCaptured = regexp.MustCompile(`^Team "(Red|Blue)" triggered "pointcaptured" \(cp "(\d+)"\) \(cpname "(.+?)"\) \(numcappers "(\d+)"\)(.*)$`)

	rxRoundStart    = regexp.MustCompile(`^World triggered "Round_Start"$`)
	rxRoundWin      = regexp.MustCompile(`^World triggered "Round_Win" \(winner "(Red|Blue)"\)$`)
	rxRoundOvertime = regexp.MustCompile(`^World triggered "Round_Overtime"$`)
	rxRoundLength   = regexp.MustCompile(`^World triggered "Round_Length" \(seconds "(.+?)"\)$`)
	rxTeamScore     = regexp.MustCompile(`^Team "(Red|Blue)" current score "(\d+)" with "(\d+)" players$`)
	rxTeamFinal     = regexp.MustCompile(`^Team "(Red|Blue)" final score "(\d+)" with "(\d+)" players$`)
	rxGameOver      = regexp.MustCompile(`^World triggered "Game_Over" reason "(.+?)"$`)
	rxPaused        = regexp.MustCompile(`^World triggered "Game_Paused"$`)
	rxUnpaused      = regexp.MustCompile(`^World triggered "Game_Unpaused"$`)
	rxCVar          = regexp.MustCompile(`^"([^"]+)" = "([^"]+)"`)
	rxServerCVar    = regexp.MustCompile(`^server_cvar:\s+"([^"]+)"\s+"([^"]+)"$`)

	rxDisconnected = regexp.MustCompile(`^disconnected \(reason "([^"]+)"\)$`)

	rxDamageValue = regexp.MustCompile(`\(damage "(\d+)"\)`)
	rxHealing     = regexp.MustCompile(`\(healing "(\d+)"\)`)
)

func parsePlayer(matches []string) Player {
	if len(matches) < 5 {
		return Player{}
	}
	return Player{
		Name:    matches[1],
		UserID:  matches[2],
		SteamID: matches[3],
		Team:    parseTeam(matches[4]),
	}
}

func parseDamageValue(body string) int {
	if m := rxDamageValue.FindStringSubmatch(body); m != nil {
		damage, _ := strconv.Atoi(m[1])
		return damage
	}
	return 0
}

const teamKey = "team"

// Parse classifies a raw Source engine log line into an Event.
// The bool return is false when the line is not recognized.
func Parse(line string) (Event, bool) {
	if matches := rxMapChange.FindStringSubmatch(line); matches != nil {
		return Event{Type: EventMapChange, Data: map[string]string{"map": matches[1]}}, true
	}

	dm := rxDatePrefix.FindStringSubmatch(line)
	if dm == nil {
		return Event{}, false
	}
	timestamp := parseTime(dm[1], dm[2])
	body := line[len(dm[0]):]

	if pm := rxPlayerPrefix.FindStringSubmatch(body); pm != nil {
		source := parsePlayer(pm)
		rest := body[len(pm[0]):]

		if matches := rxChatTeam.FindStringSubmatch(rest); matches != nil {
			return Event{Type: EventChatTeam, Timestamp: timestamp, Source: source, Data: map[string]string{"message": matches[1]}}, true
		}
		if matches := rxChat.FindStringSubmatch(rest); matches != nil {
			return Event{Type: EventChat, Timestamp: timestamp, Source: source, Data: map[string]string{"message": matches[1]}}, true
		}
		if matches := rxJoined.FindStringSubmatch(rest); matches != nil {
			source.Team = parseTeam(matches[1])
			return Event{Type: EventJoinedTeam, Timestamp: timestamp, Source: source, Data: map[string]string{teamKey: matches[1]}}, true
		}
		if matches := rxClass.FindStringSubmatch(rest); matches != nil {
			return Event{Type: EventChangeClass, Timestamp: timestamp, Source: source, Data: map[string]string{"class": matches[1]}}, true
		}
		if matches := rxSpawned.FindStringSubmatch(rest); matches != nil {
			return Event{Type: EventSpawned, Timestamp: timestamp, Source: source, Data: map[string]string{"class": matches[1]}}, true
		}
		if matches := rxSuicide.FindStringSubmatch(rest); matches != nil {
			return Event{Type: EventSuicide, Timestamp: timestamp, Source: source}, true
		}
		if matches := rxKilled.FindStringSubmatch(rest); matches != nil {
			return Event{
				Type:      EventKilled,
				Timestamp: timestamp,
				Source:    source,
				Target:    parsePlayer(matches),
			}, true
		}
		if matches := rxAssist.FindStringSubmatch(rest); matches != nil {
			return Event{
				Type:      EventKillAssist,
				Timestamp: timestamp,
				Source:    source,
				Target:    parsePlayer(matches),
			}, true
		}
		if matches := rxDamageNew.FindStringSubmatch(rest); matches != nil {
			damage := parseDamageValue(matches[5])
			return Event{
				Type:      EventDamage,
				Timestamp: timestamp,
				Source:    source,
				Target:    parsePlayer(matches),
				Data:      map[string]string{"damage": strconv.Itoa(damage)},
			}, true
		}
		if matches := rxDamageOld.FindStringSubmatch(rest); matches != nil {
			return Event{
				Type:      EventDamage,
				Timestamp: timestamp,
				Source:    source,
				Data:      map[string]string{"damage": matches[1]},
			}, true
		}
		if matches := rxHealed.FindStringSubmatch(rest); matches != nil {
			healing := 0
			if hm := rxHealing.FindStringSubmatch(matches[5]); hm != nil {
				healing, _ = strconv.Atoi(hm[1])
			}
			return Event{
				Type:      EventHealed,
				Timestamp: timestamp,
				Source:    source,
				Target:    parsePlayer(matches),
				Data: map[string]string{
					"healing": strconv.Itoa(healing),
				},
			}, true
		}
		if matches := rxDisconnected.FindStringSubmatch(rest); matches != nil {
			return Event{
				Type:      EventDisconnected,
				Timestamp: timestamp,
				Source:    source,
				Data:      map[string]string{"reason": matches[1]},
			}, true
		}
		if matches := rxCaptureBlocked.FindStringSubmatch(rest); matches != nil {
			return Event{
				Type:      EventCaptureBlocked,
				Timestamp: timestamp,
				Source:    source,
				Data: map[string]string{
					"cp":       matches[1],
					"cpname":   matches[2],
					"position": matches[3],
				},
			}, true
		}
	}

	if matches := rxPointCaptured.FindStringSubmatch(body); matches != nil {
		return Event{
			Type:      EventPointCaptured,
			Timestamp: timestamp,
			Data: map[string]string{
				teamKey:       matches[1],
				"cp":          matches[2],
				"cpname":      matches[3],
				"numcappers":  matches[4],
				"cappers_raw": strings.TrimSpace(matches[5]),
			},
		}, true
	}
	if matches := rxRoundStart.FindStringSubmatch(body); matches != nil {
		return Event{Type: EventRoundStart, Timestamp: timestamp}, true
	}
	if matches := rxRoundWin.FindStringSubmatch(body); matches != nil {
		return Event{Type: EventRoundWin, Timestamp: timestamp, Data: map[string]string{"winner": matches[1]}}, true
	}
	if matches := rxRoundOvertime.FindStringSubmatch(body); matches != nil {
		return Event{Type: EventRoundOvertime, Timestamp: timestamp}, true
	}
	if matches := rxRoundLength.FindStringSubmatch(body); matches != nil {
		return Event{Type: EventRoundLength, Timestamp: timestamp, Data: map[string]string{"seconds": matches[1]}}, true
	}
	if matches := rxTeamScore.FindStringSubmatch(body); matches != nil {
		return Event{Type: EventTeamScore, Timestamp: timestamp, Data: map[string]string{"team": matches[1], "score": matches[2], "players": matches[3]}}, true
	}
	if matches := rxTeamFinal.FindStringSubmatch(body); matches != nil {
		return Event{Type: EventTeamFinalScore, Timestamp: timestamp, Data: map[string]string{"team": matches[1], "score": matches[2], "players": matches[3]}}, true
	}
	if matches := rxGameOver.FindStringSubmatch(body); matches != nil {
		return Event{Type: EventGameOver, Timestamp: timestamp, Data: map[string]string{"reason": matches[1]}}, true
	}
	if matches := rxPaused.FindStringSubmatch(body); matches != nil {
		return Event{Type: EventGamePaused, Timestamp: timestamp}, true
	}
	if matches := rxUnpaused.FindStringSubmatch(body); matches != nil {
		return Event{Type: EventGameUnpaused, Timestamp: timestamp}, true
	}
	if matches := rxCVar.FindStringSubmatch(body); matches != nil {
		return Event{Type: EventCVar, Timestamp: timestamp, Data: map[string]string{"cvar": matches[1], "value": matches[2]}}, true
	}

	if matches := rxServerCVar.FindStringSubmatch(body); matches != nil {
		return Event{Type: EventCVar, Timestamp: timestamp, Data: map[string]string{"cvar": matches[1], "value": matches[2]}}, true
	}

	return Event{Type: EventUnknown, Timestamp: timestamp}, false
}

// ClassFromString converts a class token to a PlayerClass.
func ClassFromString(s string) PlayerClass { return parseClass(s) }

// StatusSeedEvent builds a synthetic event used to seed a player from RCON status.
func StatusSeedEvent(pl Player) Event {
	return Event{Type: EventStatusSeed, Source: pl}
}

// ResetEvent builds a synthetic event that clears the entire scoreboard.
func ResetEvent() Event {
	return Event{Type: EventReset}
}

// ParseCVar parses a raw Source engine cvar line (e.g. `"mp_timelimit" = "30"`)
// without a leading log timestamp. It returns the cvar name and value.
func ParseCVar(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if m := rxCVar.FindStringSubmatch(line); m != nil {
		return m[1], m[2], true
	}
	return "", "", false
}

// FindPlayers extracts every quoted player block from s and returns them as
// Player values. Useful for parsing cappers lists and similar log payloads.
func FindPlayers(s string) []Player {
	matches := rxPlayerBlock.FindAllStringSubmatch(s, -1)
	players := make([]Player, 0, len(matches))
	for _, match := range matches {
		players = append(players, parsePlayer(match))
	}
	return players
}
