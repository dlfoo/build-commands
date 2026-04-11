package config

import (
	"fmt"
	"io"

	"github.com/dlfoo/build-commands/pkg/plugins/kpt"
	"github.com/dlfoo/build-commands/pkg/plugins/kustomize"
	"github.com/dlfoo/build-commands/pkg/types"

	yaml "gopkg.in/yaml.v3"
)

type ConfigError struct {
	err string
}

func (c ConfigError) Error() string {
	return c.err
}

var (
	BadConfigError = &ConfigError{"misconfigured configuration"}
	UnknownError   = &ConfigError{"unknown error"}
)

type ConfigFile struct {
	Builds   []*BuildConfig `yaml:"builds"`
	Profiles types.Profiles `yaml:"profiles"`
}

type BuildConfig struct {
	Name       string                     `yaml:"name"`
	Parent     string                     `yaml:"parent"`
	ImportType types.ImportType           `yaml:"import_type"`
	Kustomize  *kustomize.KustomizeConfig `yaml:"kustomize"`
	Kpt        *kpt.KptConfig             `yaml:"kpt"`
}

// ParseConfig reads from the supplied reader and returns BuildConfig(s).
func ParseConfig(r io.Reader) ([]*BuildConfig, map[string]*types.Profile, error) {
	c := new(ConfigFile)
	if err := yaml.NewDecoder(r).Decode(c); err != nil {
		if _, ok := err.(*yaml.TypeError); ok {
			return nil, nil, fmt.Errorf("%w, got error decoding config, please check configuration is correct syntax: %v", BadConfigError, err)
		}
		return nil, nil, fmt.Errorf("%w, unknown error occured: %v", UnknownError, err)
	}
	return c.Builds, c.Profiles, nil
}
