package domain

import "time"

type ServerDatabase struct {
	ID           string
	ServerID     string
	DBName       string
	DBUser       string
	DBPassEnc    string // AES-encrypted password stored in DB
	Host         string
	Port         int
	CreatedAt    time.Time
}
