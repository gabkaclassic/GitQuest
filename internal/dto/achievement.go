package dto

import (
	"time"
)

type (
	Achievement struct {
		User            string
		RuleName        string
		RuleVersion     string
		RuleDescription string
		StartRange      time.Time
		EndRange        time.Time
		Reward          int
	}

	AchievementInfo struct {
		RuleName string `json:"rule"`
		Reward   int    `json:"reward"`
	}

	AchievementsSummary struct {
		Achievements []AchievementInfo `json:"achievements"`
		RewardSum    int               `json:"rewardSum"`
	}
)
