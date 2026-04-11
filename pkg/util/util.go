package util

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/dlfoo/build-commands/pkg/types"
)

func GetProfiles(p types.Profiles, profileSets ...[]string) types.Profiles {
	profiles := make(map[string]*types.Profile)
	for _, set := range profileSets {
		for _, name := range set {
			if prof, ok := p[name]; ok {
				profiles[name] = prof
			}
		}
	}
	return profiles
}

func GetProfilesCommands(p types.Profiles, profileSets ...[]string) []*types.BasicCommand {
	cmds := []*types.BasicCommand{}
	profiles := GetProfiles(p, profileSets...)
	for _, profile := range profiles {
		cmds = append(cmds, profile.Commands...)
	}
	return cmds
}

func getProfileOrderedList(p types.Profiles, profileSets ...[]string) []*types.Profile {
	profiles := []*types.Profile{}
	for _, set := range profileSets {
		for _, profileName := range set {
			if profile, ok := p[profileName]; ok {
				profiles = append(profiles, profile)
			}
		}
	}

	return profiles
}

func GetProfileEnvs(p types.Profiles, profileSets ...[]string) []string {
	ordered := getProfileOrderedList(p, profileSets...)
	envMapList := []map[string]string{}
	for _, profile := range ordered {
		envMapList = append(envMapList, profile.Envs)
	}
	return EvaluateEnvs(envMapList...)
}

func EvaluateEnvs(envs ...map[string]string) []string {
	finalStrings := []string{}
	toBeEval := []map[string]string{}
	osEnvs := make(map[string]string)
	for _, env := range os.Environ() {
		spl := strings.SplitN(env, "=", 2)
		if len(spl) != 2 {
			log.Fatalf("os environ %s, was formatted incorrectly", env)
		}
		osEnvs[spl[0]] = spl[1]
	}
	toBeEval = append(toBeEval, osEnvs)
	toBeEval = append(toBeEval, envs...)

	final := ParseEnvs(toBeEval...)
	for k, v := range final {
		finalStrings = append(finalStrings, fmt.Sprintf("%s=%s", k, v))
	}
	return finalStrings
}

func ParseEnvs(e ...map[string]string) map[string]string {
	m := make(map[string]string)
	for _, set := range e {
		for k, v := range set {
			upper := strings.ToUpper(k)
			if v == "" {
				delete(m, upper)
				continue
			}
			m[upper] = v
		}
	}
	return m
}

func parseArg(arg string) (string, string) {
	if strings.HasPrefix(arg, "-") {
		equal := strings.Index(arg, "=")
		space := strings.Index(arg, " ")
		if equal < space {
			split := strings.SplitN(arg, "=", 2)
			if len(split) == 2 {
				return split[0], split[1]
			}
			split = strings.SplitN(arg, " ", 2)
			if len(split) == 2 {
				return split[0], split[1]
			}
		}
		if equal > space {
			split := strings.SplitN(arg, " ", 2)
			if len(split) == 2 {
				return split[0], split[1]
			}
			split = strings.SplitN(arg, "=", 2)
			if len(split) == 2 {
				return split[0], split[1]
			}
		}

	}
	return arg, ""
}

// ParseArgs takes a list of a list of args and returns a deduplicated single list of args.
// If there are duplicate arguments the values in the []string of args at the end of the [][]string are chosen.
// for example [][]string{[]string{"--foo", "true", "--bar", "true"}, []string{"--foo", "false"}} would return []string{"--foo", "false", "--bar", "true"}
func ParseArgs(a ...[]string) []string {
	argMap := map[string]string{}
	order := []string{}
	for _, argList := range a {
		for _, arg := range argList {
			a, b := parseArg(arg)
			argMap[a] = b
			found := false
			for _, orderEntry := range order {
				if a == orderEntry {
					found = true
				}
			}
			if !found {
				order = append(order, a)
			}
		}
	}
	argList := []string{}
	for _, orderEntry := range order {
		//for k, v := range argMap {
		k := orderEntry
		v := argMap[orderEntry]
		if v != "" {
			argList = append(argList, fmt.Sprintf("%s %s", k, v))
			continue
		}
		argList = append(argList, k)
	}
	return argList
}

func ParseProfiles(profileSet ...[]string) []string {
	final := make(map[string]bool)
	finalList := []string{}
	for _, profiles := range profileSet {
		for _, p := range profiles {
			final[strings.ToLower(p)] = false
		}
	}
	for k, _ := range final {
		finalList = append(finalList, k)
	}
	return finalList
}

// Merge generic options from parent, if args or profiles contain any value the whole
// parent slice will be overridden.
func MergeGenericOptionsFromParent(a, parent types.GenericOptions, t types.ImportType) *types.GenericOptions {
	newOptions := new(types.GenericOptions)
	if t == types.ImportTypeMerge {
		newOptions.Args = ParseArgs(parent.Args, a.Args)
		newOptions.Profiles = ParseProfiles(parent.Profiles, a.Profiles)
		return newOptions
	}
	newOptions.Args = append(newOptions.Args, parent.Args...)
	newOptions.Profiles = append(newOptions.Profiles, parent.Profiles...)
	if len(a.Args) > 0 {
		newOptions.Args = a.Args
	}
	if len(a.Profiles) > 0 {
		newOptions.Profiles = a.Profiles
	}
	return newOptions
}
