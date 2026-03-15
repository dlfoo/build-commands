package types

import "os/exec"

type CommandRunMode string

const (
	RunBefore CommandRunMode = "before"
	RunWhile  CommandRunMode = "while"
	RunAfter  CommandRunMode = "after"
)

type BasicCommand struct {
	Command string         `yaml:"command"`
	Args    []string       `yaml:"args"`
	Mode    CommandRunMode `yaml:"mode"`
	Buffer  string         `yaml:"buffer"`
	Timeout string         `yaml:"timeout"`
}

type GenericOptions struct {
	Args     []string `yaml:"args"`
	Profiles []string `yaml:"profiles"`
}

type BuildCommand struct {
	ID  string
	Cmd *exec.Cmd
}

type BuildCommandSet struct {
	PluginID      string
	Cmd           *BuildCommand
	OperatingCmds []*BasicCommand
}
