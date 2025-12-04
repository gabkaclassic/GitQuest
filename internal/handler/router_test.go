package handler

import (
	"testing"

	"github.com/gabkaclassic/GitQuest/internal/config"
	"github.com/stretchr/testify/assert"
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
				Ratelimit:          &config.Ratelimit{},
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
				Ratelimit:          &config.Ratelimit{},
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
				Ratelimit:          &config.Ratelimit{},
			},
			expectErr:    true,
			errSubstring: "achievement handler is nil",
		},
		{
			name: "nil ratelimit config",
			cfg: &RouterConfiguration{
				EventHandler:       &EventHandler{},
				RuleHandler:        &RuleHandler{},
				AchievementHandler: &AchievementHandler{},
				Ratelimit:          nil,
			},
			expectErr:    true,
			errSubstring: "ratelimit config is nil",
		},
		{
			name: "ok",
			cfg: &RouterConfiguration{
				EventHandler:       &EventHandler{},
				RuleHandler:        &RuleHandler{},
				AchievementHandler: &AchievementHandler{},
				Ratelimit:          &config.Ratelimit{},
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
