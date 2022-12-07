package task

import (
	"crypto/sha256"
	"fmt"
)

// RunnerData centralizes the necessary attributes for generating the filename
type RunnerData struct {
	ShortToken string
	ProjectURL string
	PipelineID int64
	JobID      int64
}

// GenerateFilename returns a statically generated filename considering some environment parameters
func GenerateFilename(data RunnerData) string {
	longID := fmt.Sprintf(
		"%s-%d-%d",
		data.ProjectURL,
		data.PipelineID,
		data.JobID,
	)
	return fmt.Sprintf("%s-%x", data.ShortToken, sha256.Sum256([]byte(longID)))
}
