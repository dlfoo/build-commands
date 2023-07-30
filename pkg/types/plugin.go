package types

import "context"

type Plugin interface {
	Verify(string) error
	Commands(context.Context, string, Profiles) []*BuildCommandSet
	ID() string
}
