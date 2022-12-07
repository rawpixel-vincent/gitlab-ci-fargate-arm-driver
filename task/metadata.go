// Package task centralizes code related to managing tasks in the application
package task

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/encoding"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/fs"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/runner"
)

// MetadataManager represents a repository to store temporary data related to task information
type MetadataManager interface {
	// Persist stores the desired data
	Persist(data Data) error

	// Get fetches existing data from the metadata storage
	Get() (Data, error)

	// Clear removes existing data
	Clear() error
}

// Data centralizes attributes that should be persisted in the metadata storage
type Data struct {
	TaskARN     string
	ContainerIP string
	PrivateKey  []byte
}

type fsMetadataManager struct {
	logger    logging.Logger
	fs        fs.FS
	encoder   encoding.Encoder
	directory string
	filename  string
}

// NewMetadataManager is a constructor for the concrete type of the MetadataManager interface
func NewMetadataManager(logger logging.Logger, filesDir string) MetadataManager {
	manager := new(fsMetadataManager)

	manager.logger = logger
	manager.fs = fs.NewOS()
	manager.encoder = encoding.NewJSON()
	manager.directory = filesDir

	runnerAdapter := runner.GetAdapter()
	runnerData := RunnerData{
		ShortToken: runnerAdapter.ShortToken(),
		ProjectURL: runnerAdapter.ProjectURL(),
		PipelineID: runnerAdapter.PipelineID(),
		JobID:      runnerAdapter.JobID(),
	}
	manager.filename = GenerateFilename(runnerData)

	return manager
}

func (f *fsMetadataManager) Persist(data Data) error {
	f.logger.Debug("[Persist] Will persist metadata")

	buf := new(bytes.Buffer)
	err := f.encoder.Encode(data, buf)
	if err != nil {
		return fmt.Errorf("encoding data to JSON: %w", err)
	}

	err = f.fs.WriteFile(f.getFullFilePath(), buf.Bytes(), 0600)
	if err != nil {
		return fmt.Errorf("writing file %q: %w", f.getFullFilePath(), err)
	}

	f.logger.Debug("[Persist] Metadata was persisted")

	return nil
}

func (f *fsMetadataManager) getFullFilePath() string {
	return filepath.Join(f.directory, fmt.Sprintf("%s.json", f.filename))
}

func (f *fsMetadataManager) Get() (Data, error) {
	f.logger.Debug("[Get] Will get existing metadata")

	data := Data{}

	err := f.errIfFileNotExists()
	if err != nil {
		return data, fmt.Errorf("trying to access file %q: %w", f.getFullFilePath(), err)
	}

	fileContent, err := f.fs.ReadFile(f.getFullFilePath())
	if err != nil {
		return data, fmt.Errorf("reading file %q: %w", f.getFullFilePath(), err)
	}

	err = f.encoder.Decode(bytes.NewBuffer(fileContent), &data)
	if err != nil {
		return data, fmt.Errorf("decoding JSON: %w", err)
	}

	f.logger.Debug("[Get] Metadata was fetched")

	return data, nil
}

func (f *fsMetadataManager) errIfFileNotExists() error {
	exists, err := f.fs.Exists(f.getFullFilePath())
	if err != nil {
		return err
	}

	if !exists {
		return os.ErrNotExist
	}

	return nil
}

func (f *fsMetadataManager) Clear() error {
	f.logger.Debug("[Clear] Will delete existing metadata")

	err := f.errIfFileNotExists()
	if err != nil {
		return fmt.Errorf("trying to access file %q: %w", f.getFullFilePath(), err)
	}

	err = f.fs.Remove(f.getFullFilePath())
	if err != nil {
		return fmt.Errorf("deleting file %q: %w", f.getFullFilePath(), err)
	}

	f.logger.Debug("[Clear] Metadata was deleted")

	return nil
}
