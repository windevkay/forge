package workflow

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeTempFile(t *testing.T, content string) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")

	err := os.WriteFile(tmpFile, []byte(content), 0600)
	require.NoError(t, err)

	return tmpFile
}

//nolint:funlen // Test function with many test cases
func TestNewStoreFromFile(t *testing.T) {
	tests := []struct {
		name          string
		yamlContent   string
		expectError   bool
		expectedSteps map[string]int // workflow name → expected step count
	}{
		{
			name: "valid yaml with two workflows",
			yamlContent: `
workflows:
  workflow1:
    - step1:
        name: workflow1_step1
        retryafter: 5m
    - step2:
        name: workflow1_step2
        retryafter: 10m
  workflow2:
    - step1:
        name: workflow2_step1
        retryafter: 7m
    - step2:
        name: workflow2_step2
        retryafter: 5m
`,
			expectError: false,
			expectedSteps: map[string]int{
				"workflow1": 2,
				"workflow2": 2,
			},
		},
		{
			name: "invalid yaml",
			yamlContent: `
workflows
  workflow1:
    - step1:
      name: bad_yaml
`,
			expectError: true,
		},
		{
			name:          "empty file",
			yamlContent:   "",
			expectError:   false,
			expectedSteps: map[string]int{
				// No workflows at all
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := writeTempFile(t, tt.yamlContent)

			store, err := NewConfigStoreFromFile(filePath)
			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, store)

			workflows := store.GetWorkflows()
			require.Equal(t, len(tt.expectedSteps), len(workflows))

			for wf, expectedCount := range tt.expectedSteps {
				steps := workflows[wf]
				require.Equal(t, expectedCount, len(steps))

				for _, step := range steps {
					for _, s := range step {
						require.NotEmpty(t, s.Name)
						require.Greater(t, s.RetryAfter, time.Duration(0))
					}
				}
			}
		})
	}
}

func TestNewStoreFromFile_Errors(t *testing.T) {
	t.Run("file does not exist", func(t *testing.T) {
		_, err := NewConfigStoreFromFile("/path/to/nowhere.yaml")
		require.Error(t, err)
	})

	t.Run("empty file path", func(t *testing.T) {
		_, err := NewConfigStoreFromFile("")
		require.Error(t, err)
	})
}
