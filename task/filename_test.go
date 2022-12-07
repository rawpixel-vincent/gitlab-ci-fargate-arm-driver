package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateFilename(t *testing.T) {
	tests := map[string]struct {
		shortToken       string
		projectURL       string
		pipelineID       int64
		jobID            int64
		expectedFilename string
	}{
		"Valid parameters": {
			shortToken:       "test",
			projectURL:       "http://gitlab.example.com/my/project",
			pipelineID:       1,
			jobID:            1,
			expectedFilename: "test-293ac3a36135714f5953037b893ee39de22e40d9c8622a626308e13b91fa0c3f",
		},
		"Zeroed parameters": {
			shortToken:       "",
			projectURL:       "",
			pipelineID:       0,
			jobID:            0,
			expectedFilename: "-76ce04054fc6b568295815cdadc2daf5f2b5e6847592d04a7cdd6cb6c13f06fe",
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			runnerData := RunnerData{
				ShortToken: tt.shortToken,
				ProjectURL: tt.projectURL,
				PipelineID: tt.pipelineID,
				JobID:      tt.jobID,
			}
			filename := GenerateFilename(runnerData)

			assert.Equal(t, tt.expectedFilename, filename, "wrong identifier calculated")
		})
	}
}
