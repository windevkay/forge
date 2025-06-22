// Package workflows provides functionality for managing and executing workflow configurations.
//
// This package handles the parsing and storage of YAML-based workflow definitions.
// Workflows are composed of named steps that can include retry mechanisms with
// configurable delays and retry URLs.
//
// Key components:
//   - ConfigStore: Manages workflow configurations loaded from YAML files
//   - Workflow: Represents a sequence of named steps
//   - Step: Individual workflow step with retry configuration
//
// The package supports loading workflow configurations from YAML files with the
// following structure:
//
//	workflows:
//	  example-workflow:
//	    - step1:
//	        name: "First Step"
//	        retryafter: "5s"
//	        retryurl: "https://example.com/retry"
//	    - step2:
//	        name: "Second Step"
//	        retryafter: "10s"
//	        retryurl: "https://example.com/retry2"
//
// Example usage:
//
//	configStore, err := NewConfigStoreFromFile("workflows.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	workflows := configStore.GetWorkflows()
package workflows

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Step struct {
	Name       string        `yaml:"name"`
	RetryAfter time.Duration `yaml:"retryafter"`
	RetryURL   string        `yaml:"retryurl"`
}

type Workflow []map[string]Step

type Workflows map[string]Workflow

type Root struct {
	Workflows Workflows `yaml:"workflows"`
}

type ConfigStore struct {
	data Root
}

func NewConfigStoreFromFile(path string) (*ConfigStore, error) {
	if path == "" {
		return nil, errors.New("path cannot be empty")
	}

	// Clean and validate the file path to prevent path traversal attacks
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return nil, errors.New("path cannot contain '..' sequences")
	}

	// #nosec G304 - Path is validated above for security
	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = file.Close()
		if err != nil {
			log.Printf("encountered an error closing workflow config file: %s", err.Error())
		}
	}()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var root Root
	if err := yaml.Unmarshal(bytes, &root); err != nil {
		return nil, err
	}

	return &ConfigStore{data: root}, nil
}

func (s *ConfigStore) GetWorkflows() Workflows {
	return s.data.Workflows
}
