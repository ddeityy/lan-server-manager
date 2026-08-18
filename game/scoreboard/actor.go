package scoreboard

import (
	"lan-server-manager/game/logparse"
	"lan-server-manager/logger"
)

// Actor owns a Scoreboard on a single goroutine and emits snapshots after every
// applied event. It replaces direct mutex-based access for the live game state.
type Actor struct {
	events    chan logparse.Event
	snapshots chan Snapshot
	sb        *Scoreboard
	done      chan struct{}
}

// NewActor creates an actor with buffered input/output channels. BufferSize is
// applied to both channels to decouple the UI from bursty log traffic.
func NewActor(bufferSize int) *Actor {
	if bufferSize < 1 {
		bufferSize = 1
	}
	actor := &Actor{
		events:    make(chan logparse.Event, bufferSize),
		snapshots: make(chan Snapshot, bufferSize),
		sb:        New(),
		done:      make(chan struct{}),
	}
	go actor.loop()
	return actor
}

// Events returns the inbound channel used to feed parsed log and synthetic events.
func (a *Actor) Events() chan<- logparse.Event {
	return a.events
}

// Snapshots returns the outbound channel that emits a new Snapshot after each event.
func (a *Actor) Snapshots() <-chan Snapshot {
	return a.snapshots
}

// Stop shuts down the actor goroutine. The snapshot channel is closed afterwards.
func (a *Actor) Stop() {
	close(a.done)
}

func (a *Actor) loop() {
	defer close(a.snapshots)
	for {
		select {
		case evt, ok := <-a.events:
			if !ok {
				return
			}
			a.sb.Apply(evt)
			a.emitSnapshot()
		case <-a.done:
			return
		}
	}
}

func (a *Actor) emitSnapshot() {
	snap := a.sb.Snapshot()
	select {
	case a.snapshots <- snap:
	default:
		// If the UI cannot keep up, drop the oldest snapshot and push the new one.
		select {
		case <-a.snapshots:
		default:
		}
		select {
		case a.snapshots <- snap:
		case <-a.done:
		}
		logger.Warnf("scoreboard actor: dropped snapshot due to UI back-pressure")
	}
}
