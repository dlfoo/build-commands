package kustomize

import (
	"build-commands/pkg/output"
	"build-commands/pkg/types"
	"build-commands/pkg/util"
	"context"
	"fmt"
	"io/fs"
	"os/exec"
	"path/filepath"
)

var (
	pluginID   = "kustomize"
	mkdirCmdID = "create-output-dir"
	buildCmdID = "build"

	o = output.NewPlugin(new(KustomizeConfig))
)

type KustomizeBuild struct {
	types.GenericOptions `yaml:",inline"`
	TemplatePath         string `yaml:"templatePath"`
	GeneratedPath        string `yaml:"generatedPath"`
	needsGeneratedPath   bool
}

type KustomizeConfig struct {
	types.GenericOptions `yaml:",inline"`
	Build                []*KustomizeBuild `yaml:"build"`
}

func (k *KustomizeConfig) ID() string {
	return pluginID
}

func (k *KustomizeConfig) Verify(currentDir string) error {
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
	for _, kb := range k.Build {
		if _, ok := foundDir[fmt.Sprintf("%s/%s", currentDir, kb.TemplatePath)]; !ok {
			return fmt.Errorf("build Path: %s/%s, was not found in dir: %s", currentDir, kb.TemplatePath, currentDir)
		}
		if _, ok := foundDir[fmt.Sprintf("%s/%s", currentDir, kb.GeneratedPath)]; !ok {
			kb.needsGeneratedPath = true
			o.Infof("generated Path: %s/%s, was not found in dir: %s, adding command to create generated directory", currentDir, kb.GeneratedPath, currentDir)
		}
		if _, ok := foundFiles[fmt.Sprintf("%s/%s/kustomization.yaml", currentDir, kb.TemplatePath)]; !ok {
			return fmt.Errorf("kustomize.yaml was not found in dir: %s", kb.TemplatePath)
		}
	}
	return nil
}

func (k *KustomizeConfig) Commands(ctx context.Context, baseDir string, profiles types.Profiles) []*types.BuildCommandSet {
	commands := []*types.BuildCommandSet{}
	for _, b := range k.Build {
		if b.needsGeneratedPath {
			commands = append(commands, &types.BuildCommandSet{PluginID: k.ID(), Cmd: &types.BuildCommand{ID: mkdirCmdID, Cmd: exec.Command("mkdir", "-p", filepath.Join(baseDir, b.GeneratedPath))}})
		}
		cmd := &types.BuildCommand{ID: buildCmdID, Cmd: exec.CommandContext(ctx, "kustomize", "build", filepath.Join(baseDir, b.TemplatePath), "-o", filepath.Join(baseDir, b.GeneratedPath))}
		cmd.Cmd.Args = append(cmd.Cmd.Args, util.ParseArgs(k.Args, b.Args)...)
		cmd.Cmd.Env = append(cmd.Cmd.Env, util.GetProfileEnvs(profiles, k.Profiles, b.Profiles)...)
		commands = append(commands, &types.BuildCommandSet{PluginID: k.ID(), Cmd: cmd, OperatingCmds: util.GetProfilesCommands(profiles, k.Profiles, b.Profiles)})
	}
	return commands
}

func (k *KustomizeConfig) MergeFromParent(parent *KustomizeConfig) *KustomizeConfig {
	final := new(KustomizeConfig)
	opts := util.MergeGenericOptionsFromParent(parent.GenericOptions, k.GenericOptions, types.ImportTypeMerge)
	final.Args = opts.Args
	final.Profiles = opts.Profiles
	final.Build = append(final.Build, parent.Build...)
	final.Build = append(final.Build, k.Build...)
	return final
}
