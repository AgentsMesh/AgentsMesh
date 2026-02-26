package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		a, b     string
		expected int
	}{
		{"equal", "1.2.3", "1.2.3", 0},
		{"less major", "1.2.3", "2.0.0", -1},
		{"greater major", "2.0.0", "1.2.3", 1},
		{"less minor", "1.1.0", "1.2.0", -1},
		{"less patch", "1.2.1", "1.2.3", -1},
		{"v prefix", "v1.2.3", "1.2.3", 0},
		{"both v prefix", "v1.2.3", "v1.2.4", -1},
		{"different length", "1.2", "1.2.1", -1},
		{"calver style", "0.1.2025042500", "0.1.2025050100", -1},
		{"calver equal", "0.1.2025042500", "0.1.2025042500", 0},
		{"calver greater", "0.1.2025050100", "0.1.2025042500", 1},
		{"empty vs version", "", "1.0.0", -1},
		{"version vs empty", "1.0.0", "", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAdaptArgsForVersion(t *testing.T) {
	rules := []VersionArgRule{
		{
			VersionBelow: "0.1.2025042500",
			Transforms: []ArgTransform{
				{
					OldFlag: "--approval-mode",
					NewFlag: "--ask-for-approval",
				},
			},
		},
	}

	t.Run("empty version - no adaptation", func(t *testing.T) {
		args := []string{"--ask-for-approval", "suggest"}
		result := AdaptArgsForVersion(args, "", rules)
		assert.Equal(t, []string{"--ask-for-approval", "suggest"}, result)
	})

	t.Run("new version - no adaptation", func(t *testing.T) {
		args := []string{"--ask-for-approval", "suggest"}
		result := AdaptArgsForVersion(args, "0.1.2025050100", rules)
		assert.Equal(t, []string{"--ask-for-approval", "suggest"}, result)
	})

	t.Run("old version - flag transformed", func(t *testing.T) {
		args := []string{"--ask-for-approval", "suggest"}
		result := AdaptArgsForVersion(args, "0.1.2025040100", rules)
		assert.Equal(t, []string{"--approval-mode", "suggest"}, result)
	})

	t.Run("old version with value mapping", func(t *testing.T) {
		rulesWithValueMap := []VersionArgRule{
			{
				VersionBelow: "2.0.0",
				Transforms: []ArgTransform{
					{
						OldFlag: "--old-flag",
						NewFlag: "--new-flag",
						ValueMap: map[string]string{
							"new-val": "old-val",
						},
					},
				},
			},
		}
		args := []string{"--new-flag", "new-val"}
		result := AdaptArgsForVersion(args, "1.5.0", rulesWithValueMap)
		assert.Equal(t, []string{"--old-flag", "old-val"}, result)
	})

	t.Run("mixed args - only matching flags transformed", func(t *testing.T) {
		args := []string{"--model", "gpt-4", "--ask-for-approval", "suggest"}
		result := AdaptArgsForVersion(args, "0.1.2025040100", rules)
		assert.Equal(t, []string{"--model", "gpt-4", "--approval-mode", "suggest"}, result)
	})

	t.Run("no matching rules", func(t *testing.T) {
		args := []string{"--some-flag", "value"}
		result := AdaptArgsForVersion(args, "0.1.2025040100", rules)
		assert.Equal(t, []string{"--some-flag", "value"}, result)
	})

	t.Run("flag without value", func(t *testing.T) {
		args := []string{"--ask-for-approval"}
		result := AdaptArgsForVersion(args, "0.1.2025040100", rules)
		assert.Equal(t, []string{"--approval-mode"}, result)
	})
}
