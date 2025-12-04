package handler

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetupRouter(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *RouterConfiguration
		expectErr    bool
		errSubstring string
	}{
		{
			name: "nil event handler",
			cfg: &RouterConfiguration{
				EventHandler:       nil,
				RuleHandler:        &RuleHandler{},
				AchievementHandler: &AchievementHandler{},
			},
			expectErr:    true,
			errSubstring: "event handler is nil",
		},
		{
			name: "nil rule handler",
			cfg: &RouterConfiguration{
				EventHandler:       &EventHandler{},
				RuleHandler:        nil,
				AchievementHandler: &AchievementHandler{},
			},
			expectErr:    true,
			errSubstring: "rule handler is nil",
		},
		{
			name: "nil achievement handler",
			cfg: &RouterConfiguration{
				EventHandler:       &EventHandler{},
				RuleHandler:        &RuleHandler{},
				AchievementHandler: nil,
			},
			expectErr:    true,
			errSubstring: "achievement handler is nil",
		},
		{
			name: "ok",
			cfg: &RouterConfiguration{
				EventHandler:       &EventHandler{},
				RuleHandler:        &RuleHandler{},
				AchievementHandler: &AchievementHandler{},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := SetupRouter(tt.cfg)

			if tt.expectErr {
				assert.Nil(t, r)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errSubstring)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, r)
		})
	}
}
