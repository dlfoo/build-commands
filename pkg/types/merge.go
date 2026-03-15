package types

type ImportType string

var (
	ImportTypeMerge     ImportType = "merge"
	ImportTypeOverwrite ImportType = "overwrite"
)
