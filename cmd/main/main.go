package main

import (
	"build-commands/pkg/command"
	"build-commands/pkg/config"
	"build-commands/pkg/types"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	//buildNames      = flag.String("builds", "", "Only build commands for the specified builds, comma separated")
	specifiedBuilds map[string]bool

	buildNames  string
	output      string
	execCommand bool
	help        bool
	mySet       *flag.FlagSet
)

func init() {
	mySet = flag.NewFlagSet("", flag.ExitOnError)

	mySet.StringVar(&buildNames, "builds", "", "Only build commands for the specified builds, comma separated")
	mySet.StringVar(&output, "output", "", "write commands to the specified file instead of stdout")
	mySet.BoolVar(&execCommand, "exec", false, "execute commands on the machine")
	mySet.BoolVar(&help, "help", false, "show example config file")
	if len(os.Args) > 2 {
		if err := mySet.Parse(os.Args[2:]); err != nil {
			log.Print(err)
			os.Exit(1)
		}
	}
	if len(os.Args) <= 2 && strings.HasPrefix(os.Args[1], "--help") {
		if err := mySet.Parse(os.Args[1:]); err != nil {
			log.Print(err)
			os.Exit(1)
		}
	}
}

func main() {
	//flag.Parse()
	ctx := context.Background()

	if help {
		f := `
        builds:
        - name: prod
          kustomize:
            args: []
            profiles:
            - remote
            - local
            build:
            - templatePath: apps/us-west-a/prod
              generatedPath: generated/us-west-a/prod
          kpt:
            apply:
            - path: generated/us-west-a/prod
        - name: dev
          kustomize:
            profiles:
            - local
            args:
              - --my-arg before-stuff
            build:
            - templatePath: apps/us-west-a/prod
              generatedPath: generated/us-west-a/prod
              profiles:
              - remote
              args:
                - --my-arg1 stuff
        - name: from-template
          parent: dev
          import_type: merge
          kustomize:
            profiles:
            - local
            args:
              - --my-child-arg stuffstuff
            build:
            - templatePath: apps/us-west-a/prod
              generatedPath: generated/us-west-a/prod
              profiles:
              - remote
              args:
                - --my-arg2 stuff
        - name: from-template2
          parent: from-template
          import_type: merge
          kustomize:
            profiles:
            - local
            args:
              - --my-child-arg stuffstuff
            build:
            - templatePath: apps/us-west-a/prod
              generatedPath: generated/us-west-a/prod
              profiles:
              - remote
              args:
                - --my-arg3 stuff
        
        
        profiles:
          local:
            commands:
            - command: "sleep"
              args: ["60"]
              mode: "while"
              buffer: "2s"
              timeout: "30s"
            envs:
              "my_var": "my-value"
              "another_var": "another-value"
          remote:
            envs:
              "my_var": "my-value-remote"
              "remote_var": "another-remote-value"`
		cf := new(config.ConfigFile)
		if err := yaml.Unmarshal([]byte(f), cf); err != nil {
			log.Print(err)
			os.Exit(1)
		}
		b, err := yaml.Marshal(cf)
		if err != nil {
			log.Print(err)
			os.Exit(1)
		}
		fmt.Print(string(b))
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		fmt.Print("Missing build directory argument, please try again.\n")
		os.Exit(1)
	}

	if buildNames == "" {
		log.Print("'--builds' flag shouldn't be nil")
		os.Exit(1)
	}
	args := os.Args[1:]
	buildDir := args[0]
	currentDir, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}

	if buildDir == "." {
		buildDir = currentDir
	}

	if stat, err := os.Stat(buildDir); err == nil {
		if stat.IsDir() {
			buildDir = fmt.Sprintf("%s/build-commands.yaml", buildDir)
		}
		fmt.Printf("using: %s\n", buildDir)
	}

	if buildNames != "" {
		specifiedBuilds = make(map[string]bool)
		for _, b := range strings.Split(buildNames, ",") {
			specifiedBuilds[b] = true
		}
		if len(specifiedBuilds) == 0 {
			log.Fatalf("--builds command was misconfigured, please check and try again")
		}
	}
	if !filepath.IsAbs(buildDir) {
		buildDir = filepath.Join(currentDir, buildDir)
	}
	builds, profiles, err := command.GetBuilds(buildDir)
	if err != nil {
		log.Fatal(err)
	}
	outputFile := os.Stdout
	if output != "" {
		if !filepath.IsAbs(output) {
			output = filepath.Join(currentDir, output)
		}
		f, err := os.Create(output)
		if err != nil {
			log.Fatal(err)
		}
		if strings.Contains(output, ".sh") {
			fmt.Fprintln(f, "#!/bin/bash")
		}
		outputFile = f
		// for _, command := range b.GetCommands(buildDir) {
		// 	fmt.Fprintln(f, command)
		// }
		defer f.Close()
	}
	for _, b := range builds {
		if len(specifiedBuilds) > 0 {
			if _, ok := specifiedBuilds[b.Name()]; !ok {
				continue
			}
		}

		if err := command.Verify(buildDir, b); err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(outputFile, "# Build: %s\n", b.Name())

		sets := b.GetCommands(filepath.Dir(buildDir), profiles)
		for _, set := range sets {
			fmt.Fprintf(outputFile, "## Plugin: %s\n", set.PluginID)
			fmt.Fprintf(outputFile, "## Command: %s\n", set.Cmd.ID)
			newctx, cancel := context.WithCancel(ctx)
			if err := command.ExecuteCommands(newctx, types.RunBefore, set, outputFile); err != nil {
				cancel()
				break
			}
			go func() {
				if err := command.ExecuteCommands(newctx, types.RunWhile, set, outputFile); err != nil {
					cancel()
				}
			}()

			fmt.Fprintf(outputFile, "[%s][%s] %s\n", set.PluginID, set.Cmd.ID, set.Cmd.Cmd)
			if execCommand {
				c := set.Cmd.Cmd

				stdout, err := c.StdoutPipe()
				if err != nil {
					fmt.Fprint(outputFile, err)
				}

				stderr, err := c.StderrPipe()
				if err != nil {
					fmt.Fprint(outputFile, err)
				}

				if err := c.Run(); err != nil {
					fmt.Fprintf(outputFile, "[%s][%s] Non zero error code returned\n", set.PluginID, set.Cmd.ID)
					fmt.Fprintf(outputFile, "[%s][%s] ## Output ##\n", set.PluginID, set.Cmd.ID)
					io.Copy(outputFile, stdout)
					io.Copy(outputFile, stderr)
					fmt.Fprintf(outputFile, "[%s][%s] ## Exiting ##\n", set.PluginID, set.Cmd.ID)
					cancel()
					break
				}
				fmt.Fprintf(outputFile, "[%s][%s] ## Output ##\n", set.PluginID, set.Cmd.ID)
				io.Copy(outputFile, stdout)
				fmt.Fprintf(outputFile, "[%s][%s] ## Done ##\n", set.PluginID, set.Cmd.ID)
			}
			if err := command.ExecuteCommands(newctx, types.RunAfter, set, outputFile); err != nil {
				cancel()
				break
			}
			cancel()
		}
	}
	log.Print("done")
}
