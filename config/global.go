package config

import (
	"fmt"
	"io/ioutil"

	"github.com/BurntSushi/toml"
)

type Global struct {
	LogLevel  string
	LogFile   string
	LogFormat string

	Fargate      Fargate
	TaskMetadata TaskMetadata
	SSH          SSH
}

type Fargate struct {
	Cluster         string
	EnablePublicIP  bool
	PlatformVersion string
	Region          string
	Subnet          string
	SecurityGroup   string
	TaskDefinition  string
}

type TaskMetadata struct {
	Directory string
}

type SSH struct {
	Username string
	Port     int
}

func LoadFromFile(file string) (Global, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return Global{}, fmt.Errorf("couldn't read configuration file %q: %w", file, err)
	}

	var cfg Global

	err = toml.Unmarshal(data, &cfg)
	if err != nil {
		return Global{}, fmt.Errorf("couldn't parse TOML content of the configuration file: %w", err)
	}

	return cfg, nil
}
