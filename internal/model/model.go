package model

type Event struct {
	Id string `json:"id"`
	// ISO formatted start time of the event.
	// If the event is live, this is "LIVE" (due to a limitation in knowing the start time while the event is active).
	StartTime string  `json:"start_time"`
	Fights    []Fight `json:"fights"`
}

type Fight struct {
	Fighters []string `json:"fighters"`
	Winner   string   `json:"winner,omitempty"`
}
