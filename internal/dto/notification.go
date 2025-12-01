package dto

import "github.com/google/uuid"

type Notification struct {
	ID       uuid.UUID `json:"id"`
	User     string    `json:"user"`
	Reward   int       `json:"reward"`
	RuleName string    `json:"achievement"`
}
