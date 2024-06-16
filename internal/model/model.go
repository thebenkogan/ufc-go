package model

import "time"

type Event struct {
	Id string `json:"id"`
	// ISO formatted start time of the event.
	// If the event is live, this is "LIVE" (due to a limitation in knowing the start time while the event is active).
	StartTime string  `json:"start_time"`
	Fights    []Fight `json:"fights"`
}

func (e *Event) HasStarted() bool {
	if e.StartTime == "LIVE" {
		return true
	}
	t, _ := time.Parse(time.RFC3339, e.StartTime)
	return time.Now().After(t)
}

func (e *Event) IsFinished() bool {
	for _, fight := range e.Fights {
		if fight.Winner == "" {
			return false
		}
	}
	return true
}

type Fight struct {
	Fighters []string `json:"fighters"`
	Winner   string   `json:"winner,omitempty"`
}
