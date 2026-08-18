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
	ts, err := time.ParseInLocation("01/02/2006 - 15:04:05", date+" - "+t, time.Local)
	if err != nil {
		return time.Time{}
	}
	return ts
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

	rxDisconnected = regexp.MustCompile(`^disconnected \(reason "([^"]+)"\)$`)

	rxDamageValue = regexp.MustCompile(`\(damage "(\d+)"\)`)
	rxWeapon      = regexp.MustCompile(`\(weapon "(\S+)"\)`)
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

func parseDamageBody(body string) (damage int, weapon string) {
	if m := rxDamageValue.FindStringSubmatch(body); m != nil {
		damage, _ = strconv.Atoi(m[1])
	}
	if m := rxWeapon.FindStringSubmatch(body); m != nil {
		weapon = m[1]
	}
	return damage, weapon
}

// Parse classifies a raw Source engine log line into an Event.
// The bool return is false when the line is not recognized.
func Parse(line string) (Event, bool) {
	if m := rxMapChange.FindStringSubmatch(line); m != nil {
		return Event{Type: EventMapChange, Data: map[string]string{"map": m[1]}}, true
	}

	dm := rxDatePrefix.FindStringSubmatch(line)
	if dm == nil {
		return Event{}, false
	}
	ts := parseTime(dm[1], dm[2])
	body := line[len(dm[0]):]

	if pm := rxPlayerPrefix.FindStringSubmatch(body); pm != nil {
		source := parsePlayer(pm)
		rest := body[len(pm[0]):]

		if m := rxChatTeam.FindStringSubmatch(rest); m != nil {
			return Event{Type: EventChatTeam, Timestamp: ts, Source: source, Data: map[string]string{"message": m[1]}}, true
		}
		if m := rxChat.FindStringSubmatch(rest); m != nil {
			return Event{Type: EventChat, Timestamp: ts, Source: source, Data: map[string]string{"message": m[1]}}, true
		}
		if m := rxJoined.FindStringSubmatch(rest); m != nil {
			source.Team = parseTeam(m[1])
			return Event{Type: EventJoinedTeam, Timestamp: ts, Source: source, Data: map[string]string{"team": m[1]}}, true
		}
		if m := rxClass.FindStringSubmatch(rest); m != nil {
			return Event{Type: EventChangeClass, Timestamp: ts, Source: source, Data: map[string]string{"class": m[1]}}, true
		}
		if m := rxSpawned.FindStringSubmatch(rest); m != nil {
			return Event{Type: EventSpawned, Timestamp: ts, Source: source, Data: map[string]string{"class": m[1]}}, true
		}
		if m := rxSuicide.FindStringSubmatch(rest); m != nil {
			return Event{Type: EventSuicide, Timestamp: ts, Source: source, Data: map[string]string{"position": m[1]}}, true
		}
		if m := rxKilled.FindStringSubmatch(rest); m != nil {
			return Event{
				Type:      EventKilled,
				Timestamp: ts,
				Source:    source,
				Target:    parsePlayer(m),
				Data: map[string]string{
					"weapon":            m[5],
					"attacker_position": m[6],
					"victim_position":   m[7],
				},
			}, true
		}
		if m := rxAssist.FindStringSubmatch(rest); m != nil {
			return Event{
				Type:      EventKillAssist,
				Timestamp: ts,
				Source:    source,
				Target:    parsePlayer(m),
				Data: map[string]string{
					"assister_position": m[5],
					"attacker_position": m[6],
					"victim_position":   m[7],
				},
			}, true
		}
		if m := rxDamageNew.FindStringSubmatch(rest); m != nil {
			damage, weapon := parseDamageBody(m[5])
			return Event{
				Type:      EventDamage,
				Timestamp: ts,
				Source:    source,
				Target:    parsePlayer(m),
				Data: map[string]string{
					"raw":    m[5],
					"damage": strconv.Itoa(damage),
					"weapon": weapon,
				},
			}, true
		}
		if m := rxDamageOld.FindStringSubmatch(rest); m != nil {
			return Event{
				Type:      EventDamage,
				Timestamp: ts,
				Source:    source,
				Data:      map[string]string{"damage": m[1]},
			}, true
		}
		if m := rxHealed.FindStringSubmatch(rest); m != nil {
			healing := 0
			if hm := rxHealing.FindStringSubmatch(m[5]); hm != nil {
				healing, _ = strconv.Atoi(hm[1])
			}
			return Event{
				Type:      EventHealed,
				Timestamp: ts,
				Source:    source,
				Target:    parsePlayer(m),
				Data: map[string]string{
					"healing": strconv.Itoa(healing),
				},
			}, true
		}
		if m := rxDisconnected.FindStringSubmatch(rest); m != nil {
			return Event{
				Type:      EventDisconnected,
				Timestamp: ts,
				Source:    source,
				Data:      map[string]string{"reason": m[1]},
			}, true
		}
		if m := rxCaptureBlocked.FindStringSubmatch(rest); m != nil {
			return Event{
				Type:      EventCaptureBlocked,
				Timestamp: ts,
				Source:    source,
				Data: map[string]string{
					"cp":       m[1],
					"cpname":   m[2],
					"position": m[3],
				},
			}, true
		}
	}

	if m := rxPointCaptured.FindStringSubmatch(body); m != nil {
		return Event{
			Type:      EventPointCaptured,
			Timestamp: ts,
			Data: map[string]string{
				"team":        m[1],
				"cp":          m[2],
				"cpname":      m[3],
				"numcappers":  m[4],
				"cappers_raw": strings.TrimSpace(m[5]),
			},
		}, true
	}
	if m := rxRoundStart.FindStringSubmatch(body); m != nil {
		return Event{Type: EventRoundStart, Timestamp: ts}, true
	}
	if m := rxRoundWin.FindStringSubmatch(body); m != nil {
		return Event{Type: EventRoundWin, Timestamp: ts, Data: map[string]string{"winner": m[1]}}, true
	}
	if m := rxRoundOvertime.FindStringSubmatch(body); m != nil {
		return Event{Type: EventRoundOvertime, Timestamp: ts}, true
	}
	if m := rxRoundLength.FindStringSubmatch(body); m != nil {
		return Event{Type: EventRoundLength, Timestamp: ts, Data: map[string]string{"seconds": m[1]}}, true
	}
	if m := rxTeamScore.FindStringSubmatch(body); m != nil {
		return Event{Type: EventTeamScore, Timestamp: ts, Data: map[string]string{"team": m[1], "score": m[2], "players": m[3]}}, true
	}
	if m := rxTeamFinal.FindStringSubmatch(body); m != nil {
		return Event{Type: EventTeamFinalScore, Timestamp: ts, Data: map[string]string{"team": m[1], "score": m[2], "players": m[3]}}, true
	}
	if m := rxGameOver.FindStringSubmatch(body); m != nil {
		return Event{Type: EventGameOver, Timestamp: ts, Data: map[string]string{"reason": m[1]}}, true
	}
	if m := rxPaused.FindStringSubmatch(body); m != nil {
		return Event{Type: EventGamePaused, Timestamp: ts}, true
	}
	if m := rxUnpaused.FindStringSubmatch(body); m != nil {
		return Event{Type: EventGameUnpaused, Timestamp: ts}, true
	}
	if m := rxCVar.FindStringSubmatch(body); m != nil {
		return Event{Type: EventCVar, Timestamp: ts, Data: map[string]string{"cvar": m[1], "value": m[2]}}, true
	}

	return Event{Type: EventUnknown, Timestamp: ts}, false
}

// ClassFromString converts a class token to a PlayerClass.
func ClassFromString(s string) PlayerClass { return parseClass(s) }

// FindPlayers extracts every quoted player block from s and returns them as
// Player values. Useful for parsing cappers lists and similar log payloads.
func FindPlayers(s string) []Player {
	var players []Player
	matches := rxPlayerBlock.FindAllStringSubmatch(s, -1)
	for _, m := range matches {
		players = append(players, parsePlayer(m))
	}
	return players
}
