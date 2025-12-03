package dto

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"gopkg.in/yaml.v3"
	"time"
)

type (
	Condition string
	TimeRange time.Duration

	Rule struct {
		Name        string    `yaml:"name" json:"name"`
		Description string    `yaml:"description" json:"description"`
		EventType   EventType `yaml:"event_type" json:"eventType"`
		Window      TimeRange `yaml:"window" json:"window"`
		Count       int       `yaml:"count" json:"count"`
		Condition   Condition `yaml:"condition" json:"condition"`
		Streak      bool      `yaml:"streak" json:"streak"`
		Reward      int       `yaml:"reward" json:"reward"`
		Version     string    `yaml:"-" json:"-"`
	}
	RuleDiff struct {
		Name       string
		OldVersion string
		NewVersion string
		RewardDiff int
	}
	ruleHash struct {
		EventType EventType `yaml:"event_type"`
		Window    TimeRange `yaml:"window"`
		Count     int       `yaml:"count"`
		Condition Condition `yaml:"condition"`
		Streak    bool      `yaml:"streak"`
		Reward    int       `yaml:"reward"`
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

func (r *Rule) UnmarshalYAML(unmarshal func(any) error) error {
	type ruleAlias struct {
		Name        string    `yaml:"name"`
		Description string    `yaml:"description"`
		EventType   EventType `yaml:"event_type"`
		Window      TimeRange `yaml:"window"`
		Count       int       `yaml:"count"`
		Condition   Condition `yaml:"condition"`
		Streak      bool      `yaml:"streak"`
		Reward      int       `yaml:"reward"`
	}

	var tmp ruleAlias
	if err := unmarshal(&tmp); err != nil {
		return err
	}

	r.Name = tmp.Name
	r.Description = tmp.Description
	r.EventType = tmp.EventType
	r.Window = tmp.Window
	r.Count = tmp.Count
	r.Condition = tmp.Condition
	r.Streak = tmp.Streak
	r.Reward = tmp.Reward

	hashStruct := ruleHash{
		EventType: r.EventType,
		Window:    r.Window,
		Count:     r.Count,
		Condition: r.Condition,
		Streak:    r.Streak,
		Reward:    r.Reward,
	}

	hash, err := computeHash(hashStruct)
	if err != nil {
		return fmt.Errorf("failed to compute version hash: %w", err)
	}

	r.Version = hash
	return nil
}

func computeHash(data any) (string, error) {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(yamlData)
	return hex.EncodeToString(hash[:]), nil
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
	var raw string
	if err := unmarshal(&raw); err != nil {
		return err
	}

	duration, err := time.ParseDuration(raw)
	if err != nil {
		return fmt.Errorf("invalid duration format '%s': %w", raw, err)
	}

	if duration > MaxTimeRange {
		return fmt.Errorf("time duration %v is more than maximum %v", duration, MaxTimeRange)
	}

	if duration < 0 {
		return fmt.Errorf("time duration %v can't be negative", duration)
	}

	*tr = TimeRange(duration)
	return nil
}

func (tr TimeRange) ToNanoseconds() int64 {
	return time.Duration(tr).Nanoseconds()
}

func FromNanoseconds(ns int64) TimeRange {
	return TimeRange(time.Duration(ns))
}

func (c *Condition) Compare(a, b int) bool {
	switch *c {
	case LT:
		return a < b
	case GT:
		return a > b
	case LTE:
		return a <= b
	case GTE:
		return a >= b
	case EQ:
		return a == b
	default:
		return false
	}
}
