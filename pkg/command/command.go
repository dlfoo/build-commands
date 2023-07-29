package command

import (
	"build-commands/pkg/config"
	"build-commands/pkg/output"
	"build-commands/pkg/types"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Build struct {
	bc       *config.BuildConfig
	profiles map[string]*types.Profile
	tools    []types.Plugin
}

func (b *Build) Name() string {
	return b.bc.Name
}

func (b *Build) GetCommands(baseDir string, profiles types.Profiles) []*types.BuildCommandSet {
	commandSets := []*types.BuildCommandSet{}
	for _, tool := range b.tools {
		commandSets = append(commandSets, tool.Commands(baseDir, profiles)...)
	}
	return commandSets
}

func GetBuilds(filename string, buildFilter map[string]bool) ([]*Build, map[string]*types.Profile, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	buildConfigs, pfs, err := config.ParseConfig(f)
	if err != nil {
		return nil, nil, err
	}
	if len(buildConfigs) == 0 {
		fmt.Printf("no builds found in %s\n", f.Name())
		os.Exit(0)
	}
	builds := []*Build{}
	buildConfigMap := make(map[string]*config.BuildConfig)
	for _, bc := range buildConfigs {
		buildConfigMap[bc.Name] = bc
	}
	for _, bc := range buildConfigs {
		if bc != nil {
			kustomizeConfig := bc.Kustomize
			commands := []types.Plugin{}
			if bc.Kustomize != nil {
				if bc.Parent != "" && bc.ImportType == types.ImportTypeMerge {
					if p, ok := buildConfigMap[bc.Parent]; ok {
						kustomizeConfig = kustomizeConfig.MergeFromParent(p.Kustomize)
						bc.Kustomize = kustomizeConfig
					}
				}
				commands = append(commands, kustomizeConfig)
			}
			kptConfig := bc.Kpt
			if bc.Kpt != nil {
				if bc.Parent != "" && bc.ImportType == types.ImportTypeMerge {
					if p, ok := buildConfigMap[bc.Parent]; ok {
						kptConfig = kptConfig.MergeFromParent(p.Kpt)
						bc.Kpt = kptConfig
					}
				}
				commands = append(commands, kptConfig)
			}
			builds = append(builds, &Build{bc: bc, profiles: pfs, tools: commands})
		}
	}

	// Verify
	for _, b := range builds {
		for _, c := range b.tools {
			if err := c.Verify(filepath.Dir(filename)); err != nil {
				return nil, nil, err
			}
		}
	}

	return builds, pfs, nil
}

// func getEnvironmentVariables(p types.Profiles) []string {
// 	vars := os.Environ()
// 	envMap := map[string]string{}
// 	for _, profile := range p {
// 		for name, value := range profile.Envs {
// 			envMap[name] = fmt.Sprintf("%s=%s", strings.ToUpper(name), value)
// 		}
// 	}
// 	for _, entry := range envMap {
// 		vars = append(vars, entry)
// 	}
// 	return vars
// }

func ExecuteCommands(ctx context.Context, m types.CommandRunMode, set *types.BuildCommandSet, outputFile *os.File) ([]*output.CommandResult, error) {
	res := []*output.CommandResult{}
	var wg sync.WaitGroup
	for _, c := range set.OperatingCmds {
		if c.Mode != m {
			continue
		}
		if c.Timeout == "" {
			c.Timeout = "30s"
		}
		timeout, err := time.ParseDuration(c.Timeout)
		if err != nil {
			return res, err
		}
		newctx, cancel := context.WithTimeout(ctx, timeout)
		cmd := exec.Command(c.Command, c.Args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return res, err
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return res, err
		}
		fmt.Fprintf(outputFile, "[%s][%s][%s] %s\n", set.PluginID, set.Cmd.ID, m, cmd.String())
		switch c.Mode {
		case types.RunBefore, types.RunAfter:
			if err := cmd.Run(); err != nil {
				cancel()
				return res, err
			}
		case types.RunWhile:
			wg.Add(1)
			go func() {
				err := cmd.Start()
				if err != nil {
					fmt.Fprintf(outputFile, "[%s][%s][%s] %s returned err upon start\n", set.PluginID, set.Cmd.ID, m, cmd.String())
					cancel()
					os.Exit(1)
				}
				go func() {
					<-newctx.Done()
					fmt.Fprintf(outputFile, "[%s][%s][%s] stopping %s\n", set.PluginID, set.Cmd.ID, m, cmd.String())
					if err := cmd.Process.Signal(os.Interrupt); err != nil {
						log.Printf("got err interrupting %v", err)
					}
				}()

				st, err := cmd.Process.Wait()
				if err != nil {
					fmt.Fprintf(outputFile, "[%s][%s][%s] %s returned %s upon exit with process state %s\n", set.PluginID, set.Cmd.ID, m, cmd.String(), err, st)
					cancel()
					os.Exit(1)
				}
				cancel()
				wg.Done()
			}()
		default:
			cancel()
			return res, fmt.Errorf("mode %q not found", m)
		}

		if c.Buffer != "" {
			d, err := time.ParseDuration(c.Buffer)
			if err != nil {
				cancel()
				return res, err
			}
			time.Sleep(d)
		}
	}
	return res, nil
}
