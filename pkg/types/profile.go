package types

type Profile struct {
	Envs     map[string]string `yaml:"envs"`
	Commands []*BasicCommand   `yaml:"commands"`
}

type Profiles map[string]*Profile
