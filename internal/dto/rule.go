package dto

import (
	"fmt"
	"time"
)

type (
	Condition string
	TimeRange time.Duration

	Rule struct {
		Name        string    `yaml:"name"`
		Description string    `yaml:"description"`
		EventType   EventType `yaml:"event_type"`
		Window      TimeRange `yaml:"window"`
		Count       int       `yaml:"count"`
		Condition   Condition `yaml:"condition"`
		Streak      bool      `yaml:"streak"`
		Reward      int       `yaml:"reward"`
		Hash        string    `yaml:"-"`
	}
)

const (
	GT  Condition = ">"
	GTE Condition = ">="
	EQ  Condition = "=="
	LT  Condition = "<"
	LTE Condition = "<="

	MaxTimeRange = 24 * 30 * time.Hour
)

var validConditions = map[Condition]struct{}{
	GT:  {},
	GTE: {},
	EQ:  {},
	LT:  {},
	LTE: {},
}

func (c *Condition) UnmarshalYAML(unmarshal func(any) error) error {
	var raw string
	if err := unmarshal(&raw); err != nil {
		return err
	}

	cond := Condition(raw)
	if _, ok := validConditions[cond]; !ok {
		return fmt.Errorf("invalid condition: %s", raw)
	}

	*c = cond
	return nil
}

func (tr *TimeRange) UnmarshalYAML(unmarshal func(any) error) error {
	var raw time.Duration
	if err := unmarshal(&raw); err != nil {
		return err
	}

	if raw > MaxTimeRange {
		return fmt.Errorf("time duration %v is more than maximum %v", raw, MaxTimeRange)
	}

	return nil
}
