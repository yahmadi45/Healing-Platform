package domain

import "time"

type Service struct {
	Name        string    `json:"name"`
	Environment string    `json:"environment"`
	Status      string    `json:"status"`
	Icon        string    `json:"icon"`
	LastCheck   time.Time `json:"last_check"`
	Recoveries  int       `json:"recoveries"`
}

type Incident struct {
	ID        string    `json:"id"`
	Service   string    `json:"service"`
	Title     string    `json:"title"`
	Detail    string    `json:"detail"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type Event struct {
	ID        string    `json:"id"`
	Service   string    `json:"service"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}

type Stats struct {
	Services      int `json:"services"`
	Healthy       int `json:"healthy"`
	OpenIncidents int `json:"open_incidents"`
	Recoveries    int `json:"recoveries"`
}
