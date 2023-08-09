package output

import (
	"build-commands/pkg/types"
	"fmt"
	"os"
)

type outputFormat string

var (
	outputFormatPlugin outputFormat = "plugin"
)

var (
	pluginTemplate = "[%s] %s"
)

func NewPlugin(p types.Plugin) *Output {
	return &Output{
		format: outputFormatPlugin,
		stdOut: os.Stdout,
		stdErr: os.Stderr,
		plugin: p,
	}
}

type Output struct {
	format outputFormat
	stdOut *os.File
	stdErr *os.File
	plugin types.Plugin
}

type CommandResultStatus string

const (
	CommandResultStatusRunning CommandResultStatus = "running"
	CommandResultStatusStopped CommandResultStatus = "stopped"
)

type CommandResult struct {
	InvocationID string
	Status       CommandResultStatus `json:"status"`
	Build        string              `json:"build"`
	Command      string              `json:"command"`
	Stdout       string              `json:"std_out"`
	Stderr       string              `json:"std_err"`
	ExitCode     int                 `json:"exit_code"`
	Pid          int                 `json:"pid"`
}

func (o *Output) Infof(format string, a ...interface{}) {
	switch o.format {
	case outputFormatPlugin:
		o.stdOut.WriteString(fmt.Sprintf(pluginTemplate, o.plugin.ID(), fmt.Sprintf(format, a...)))
	default:
		o.stdOut.WriteString(fmt.Sprintf(format, a...))
	}
	o.stdOut.WriteString("\n")
}

func (o *Output) Errorf(format string, a ...interface{}) {
	switch o.format {
	case outputFormatPlugin:
		o.stdErr.WriteString(fmt.Sprintf(pluginTemplate, o.plugin.ID(), fmt.Sprintf(format, a...)))
	default:
		o.stdErr.WriteString(fmt.Sprintf(format, a...))
	}
	o.stdOut.WriteString("\n")
}
