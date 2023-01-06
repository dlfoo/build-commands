package kpt

import (
	"build-commands/pkg/output"
	"build-commands/pkg/types"
	"build-commands/pkg/util"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
)

var (
	pluginID   = "kpt"
	initCmdID  = "package-init"
	applyCmdID = "live-apply"

	o = output.NewPlugin(new(KptConfig))
)

type KptConfig struct {
	types.GenericOptions `yaml:",inline"`
	Apply                []*KptApply `yaml:"apply"`
}

type KptApply struct {
	types.GenericOptions `yaml:",inline"`
	Path                 types.PathString `yaml:"path"`
	needInit             bool             `yaml:"_"`
}

func (k *KptConfig) ID() string {
	return pluginID
}

func (k *KptConfig) Verify(currentDir string) error {
	foundDir := map[string]bool{}
	foundFiles := map[string]bool{}

	err := filepath.Walk(currentDir, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			foundDir[path] = true
			return nil
		}
		foundFiles[path] = true
		return nil
	})
	if err != nil {
		return err
	}

	for _, c := range k.Apply {
		if _, ok := foundDir[fmt.Sprintf("%s/%s", currentDir, string(c.Path))]; !ok {
			return fmt.Errorf("generated config Path: %s/%s, was not found in dir: %s", currentDir, c.Path, currentDir)
		}
		if _, ok := foundFiles[fmt.Sprintf("%s/%s/resourcegroup.yaml", currentDir, string(c.Path))]; !ok {
			c.needInit = true
			o.Infof("Kpt resourcegroup.yaml file was not found in dir: %s, will add kpt pkg init command.", c.Path)
		}
	}
	return nil
}

func (k *KptConfig) Commands(baseDir string, profiles types.Profiles) []*types.BuildCommandSet {
	commands := []*types.BuildCommandSet{}
	for _, b := range k.Apply {
		if b.needInit {
			commands = append(commands, &types.BuildCommandSet{PluginID: k.ID(), Cmd: &types.BuildCommand{ID: initCmdID, Cmd: exec.Command("kpt", "live", "init", filepath.Join(baseDir, string(b.Path)))}})
		}
		cmd := &types.BuildCommand{
			ID:  applyCmdID,
			Cmd: exec.Command("kpt", "live", "apply", filepath.Join(baseDir, string(b.Path))),
		}
		cmd.Cmd.Args = append(cmd.Cmd.Args, util.ParseArgs(k.Args, b.Args)...)
		cmd.Cmd.Env = append(cmd.Cmd.Env, util.GetProfileEnvs(profiles, k.Profiles, b.Profiles)...)
		commands = append(commands, &types.BuildCommandSet{PluginID: k.ID(), Cmd: cmd, OperatingCmds: util.GetProfilesCommands(profiles, k.Profiles, b.Profiles)})
	}
	return commands
}

func (k *KptConfig) MergeFromParent(parent *KptConfig) *KptConfig {
	final := new(KptConfig)
	opts := util.MergeGenericOptionsFromParent(parent.GenericOptions, k.GenericOptions, types.ImportTypeMerge)
	final.Args = opts.Args
	final.Profiles = opts.Profiles
	final.Apply = append(final.Apply, parent.Apply...)
	final.Apply = append(final.Apply, k.Apply...)
	return final
}
