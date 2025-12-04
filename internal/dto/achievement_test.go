package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAchievementInfo_MarshalBinary(t *testing.T) {
	tests := []struct {
		name        string
		achievement AchievementInfo
		expected    string
	}{
		{
			name: "basic achievement",
			achievement: AchievementInfo{
				RuleName: "rule1",
				Reward:   10,
			},
			expected: `{"rule":"rule1","reward":10}`,
		},
		{
			name: "another achievement",
			achievement: AchievementInfo{
				RuleName: "rule2",
				Reward:   5,
			},
			expected: `{"rule":"rule2","reward":5}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.achievement.MarshalBinary()
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestAchievementInfo_UnmarshalBinary(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    AchievementInfo
		expectError bool
	}{
		{
			name:  "valid json",
			input: `{"rule":"rule1","reward":10}`,
			expected: AchievementInfo{
				RuleName: "rule1",
				Reward:   10,
			},
		},
		{
			name:        "invalid json",
			input:       `{not valid json}`,
			expectError: true,
		},
		{
			name:  "missing reward field",
			input: `{"rule":"rule2"}`,
			expected: AchievementInfo{
				RuleName: "rule2",
				Reward:   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ach AchievementInfo
			err := ach.UnmarshalBinary([]byte(tt.input))
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, ach)
			}
		})
	}
}

func TestAchievementInfo_MarshalAndUnmarshalBinary(t *testing.T) {
	tests := []struct {
		name        string
		achievement AchievementInfo
	}{
		{
			name: "roundtrip achievement1",
			achievement: AchievementInfo{
				RuleName: "rule1",
				Reward:   10,
			},
		},
		{
			name: "roundtrip achievement2",
			achievement: AchievementInfo{
				RuleName: "rule2",
				Reward:   5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.achievement.MarshalBinary()
			assert.NoError(t, err)

			var decoded AchievementInfo
			err = decoded.UnmarshalBinary(data)
			assert.NoError(t, err)

			assert.Equal(t, tt.achievement, decoded)
		})
	}
}
