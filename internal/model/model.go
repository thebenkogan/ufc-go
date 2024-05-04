package model

type Event struct {
	Id        string  `json:"id"`
	StartTime string  `json:"start_time"`
	Fights    []Fight `json:"fights"`
}

type Fight struct {
	Fighters []string `json:"fighters"`
	Winner   string   `json:"winner,omitempty"`
}
