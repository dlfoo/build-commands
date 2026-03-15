package util_test

import (
	"sort"
	"testing"

	"github.com/dlfoo/build-commands/pkg/types"
	"github.com/dlfoo/build-commands/pkg/util"

	"github.com/google/go-cmp/cmp"
)

func TestParseArgs(t *testing.T) {
	input := [][]string{
		{
			"--baz stuff=stuff",
			"--foo=stuff=stuff",
			"--bar false",
			"--pal true",
			"--car=stuff stuff",
			"--etc",
		},
		{
			"--pal=true",
		},
		{
			"--bar=true",
			"--pal false",
			"--etc",
		},
	}

	want := []string{
		"--baz stuff=stuff",
		"--foo stuff=stuff",
		"--bar true",
		"--pal false",
		"--car stuff stuff",
		"--etc",
	}

	got := util.ParseArgs(input...)
	//fmt.Print(got)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("got unexpected diff %s", diff)
	}

}

func TestParseEnvs(t *testing.T) {
	input := []map[string]string{
		{
			"FOO": "stuff",
			"BAR": "more-stuff",
			"baz": "false",
		},
		{
			"foo":     "different-stuff",
			"BAR":     "",
			"FOO_BAR": "diff",
		},
	}

	want := map[string]string{
		"FOO":     "different-stuff",
		"BAZ":     "false",
		"FOO_BAR": "diff",
	}

	got := util.ParseEnvs(input...)
	//fmt.Print(got)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("got unexpected diff %s", diff)
	}

}

func TestGetProfileEnvs(t *testing.T) {
	profiles := map[string]*types.Profile{
		"local": {
			Envs: map[string]string{
				"FOO": "stuff",
				"BAR": "more-stuff",
				"baz": "false",
			},
		},
		"remote": {
			Envs: map[string]string{
				"remote":  "remote-stuff",
				"foo":     "different-stuff",
				"BAR":     "",
				"FOO_BAR": "diff",
			},
		},
		"another": {
			Envs: map[string]string{
				"foo":     "different-stuff",
				"FOO_BAR": "diff",
			},
		},
	}
	input := [][]string{
		{"local", "remote"},
		{"another"},
	}

	want := []string{
		"REMOTE=remote-stuff",
		"FOO=different-stuff",
		"BAZ=false",
		"FOO_BAR=diff",
	}

	got := util.GetProfileEnvs(profiles, input...)

	sort.Strings(got)
	sort.Strings(want)
	//fmt.Print(got)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("got unexpected diff %s", diff)
	}

}
