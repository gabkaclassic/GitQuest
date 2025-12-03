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
				EventHandler: nil,
				RuleHandler:  &RuleHandler{},
			},
			expectErr:    true,
			errSubstring: "event handler is nil",
		},
		{
			name: "nil rule handler",
			cfg: &RouterConfiguration{
				EventHandler: &EventHandler{},
				RuleHandler:  nil,
			},
			expectErr:    true,
			errSubstring: "rule handler is nil",
		},
		{
			name: "ok",
			cfg: &RouterConfiguration{
				EventHandler: &EventHandler{},
				RuleHandler:  &RuleHandler{},
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
