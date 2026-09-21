package core

// CommandType represents the type of command requested by the user.
type CommandType int

const (
	Unknown CommandType = iota
	AddLabel
	RmLabel
	AddMilestone
	RmMilestone
	SelfAssign
	SelfUnassign
)

// ParsedCommand holds the command type and its accompanying value/argument.
type ParsedCommand struct {
	Type  CommandType
	Value string
}
