package models

import "time"

type Edge struct {
	ID        int64
	Name      string
	Hostname  string
	IPAddress string
	Status    string
	Version   string
	LastSeen  time.Time
	CreatedAt time.Time
}
