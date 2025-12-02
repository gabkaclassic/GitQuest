package service

import (
	"errors"
	"testing"

	"github.com/gabkaclassic/GitQuest/internal/dto"
	"github.com/gabkaclassic/GitQuest/internal/repository"
	"github.com/stretchr/testify/assert"
)

func TestNewRuleService(t *testing.T) {
	tests := []struct {
		name        string
		repository  repository.RuleRepository
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid repository",
			repository:  &repository.MockRuleRepository{},
			expectError: false,
		},
		{
			name:        "nil repository",
			repository:  nil,
			expectError: true,
			errorMsg:    "repository is nil",
		},
		{
			name:        "mock repository implementation",
			repository:  &repository.MockRuleRepository{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewRuleService(tt.repository)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, service)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, service)
				assert.IsType(t, &ruleService{}, service)

				svc := service.(*ruleService)
				assert.Equal(t, tt.repository, svc.repository)
			}
		})
	}
}

func TestRuleService_SaveAll(t *testing.T) {
	repo := new(repository.MockRuleRepository)
	service := &ruleService{repository: repo}

	rules := []dto.Rule{{}}

	t.Run("repo error", func(t *testing.T) {
		repo.ExpectedCalls = nil
		repo.On("SaveAll", rules).Return(errors.New("db fail"))

		err := service.SaveAll(rules)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "db fail")
		repo.AssertExpectations(t)
	})

	t.Run("success", func(t *testing.T) {
		repo.ExpectedCalls = nil
		repo.On("SaveAll", rules).Return(nil)

		err := service.SaveAll(rules)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRuleService_GetRulesDiffs(t *testing.T) {
	type mocks struct {
		repo *repository.MockRuleRepository
	}

	r1 := dto.Rule{Name: "r1", Version: "2", Reward: 20}
	r2 := dto.Rule{Name: "r2", Version: "1", Reward: 5}

	lastR1 := &dto.Rule{Name: "r1", Version: "1", Reward: 10}

	tests := []struct {
		name      string
		setup     func(m mocks)
		rules     []dto.Rule
		expect    []dto.RuleDiff
		expectErr string
	}{
		{
			name: "repo returns error",
			setup: func(m mocks) {
				m.repo.On("GetLastRule", "r1").
					Return(&dto.Rule{}, errors.New("db err"))
			},
			rules:     []dto.Rule{r1},
			expectErr: "db err",
		},
		{
			name: "rule reward increased add diff",
			setup: func(m mocks) {
				m.repo.On("GetLastRule", "r1").Return(lastR1, nil)
			},
			rules: []dto.Rule{r1},
			expect: []dto.RuleDiff{
				{
					Name:       "r1",
					OldVersion: "1",
					NewVersion: "2",
					RewardDiff: 10,
				},
			},
		},
		{
			name: "reward decreased ignore",
			setup: func(m mocks) {
				m.repo.On("GetLastRule", "r2").Return(&dto.Rule{
					Name:    "r2",
					Version: "0",
					Reward:  10,
				}, nil)
			},
			rules:  []dto.Rule{r2},
			expect: []dto.RuleDiff{},
		},
		{
			name: "version same ignore",
			setup: func(m mocks) {
				m.repo.On("GetLastRule", "r1").Return(&dto.Rule{
					Name:    "r1",
					Version: "2",
					Reward:  10,
				}, nil)
			},
			rules:  []dto.Rule{r1},
			expect: []dto.RuleDiff{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockObjs := mocks{
				repo: new(repository.MockRuleRepository),
			}

			tt.setup(mockObjs)

			service := &ruleService{
				repository: mockObjs.repo,
			}

			out, err := service.GetRulesDiffs(tt.rules)

			if tt.expectErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tt.expect, out)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErr)
			}

			mockObjs.repo.AssertExpectations(t)
		})
	}
}
