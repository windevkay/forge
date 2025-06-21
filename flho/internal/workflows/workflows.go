package workflows

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Step struct {
	Name       string        `yaml:"name"`
	RetryAfter time.Duration `yaml:"retryafter"`
	RetryUrl   string        `yaml:"retryurl"`
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

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
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
