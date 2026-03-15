package types

type Plugin interface {
	Verify(string) error
	Commands(string, Profiles) []*BuildCommandSet
	ID() string
}
