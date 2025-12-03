package dto

import (
	"time"
)

type Achievement struct {
	User            string
	RuleName        string
	RuleVersion     string
	RuleDescription string
	StartRange      time.Time
	EndRange        time.Time
	Reward          int
}
