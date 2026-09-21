package core

import (
	"fmt"
	"strings"
)

const (
	addLabel     = "add label "
	rmLabel      = "rm label "
	addMilestone = "add milestone "
	rmMilestone  = "rm milestone "
	selfAssign   = "self assign "
	selfUnassign = "self unassign "
)

// ParseCommand pre-processes a raw string into a structured ParsedCommand.
func ParseCommand(commandStr string) ParsedCommand {
	if after, ok := strings.CutPrefix(commandStr, addLabel); ok {
		return ParsedCommand{
			Type:  AddLabel,
			Value: after,
		}
	} else if after0, ok0 := strings.CutPrefix(commandStr, rmLabel); ok0 {
		return ParsedCommand{
			Type:  RmLabel,
			Value: after0,
		}
	} else if after1, ok1 := strings.CutPrefix(commandStr, addMilestone); ok1 {
		return ParsedCommand{
			Type:  AddMilestone,
			Value: after1,
		}
	} else if after2, ok2 := strings.CutPrefix(commandStr, rmMilestone); ok2 {
		return ParsedCommand{
			Type:  RmMilestone,
			Value: after2,
		}
	} else if after3, ok3 := strings.CutPrefix(commandStr, selfAssign); ok3 {
		return ParsedCommand{
			Type:  SelfAssign,
			Value: after3,
		}
	} else if after4, ok4 := strings.CutPrefix(commandStr, selfUnassign); ok4 {
		return ParsedCommand{
			Type:  SelfUnassign,
			Value: after4,
		}
	}

	return ParsedCommand{
		Type:  Unknown,
		Value: commandStr,
	}
}

// ExecuteCommand prints information based on the parsed command type (matching Python's utility).
func ExecuteCommand(cmd ParsedCommand) {
	switch cmd.Type {
	case AddLabel:
		fmt.Printf("Adding label: %s\n", cmd.Value)
	case RmLabel:
		fmt.Printf("Removing label: %s\n", cmd.Value)
	case AddMilestone:
		fmt.Printf("Adding milestone: %s\n", cmd.Value)
	case RmMilestone:
		fmt.Printf("Removing milestone: %s\n", cmd.Value)
	case SelfAssign:
		fmt.Printf("Self-assigning to: %s\n", cmd.Value)
	case SelfUnassign:
		fmt.Printf("Self-unassigning from: %s\n", cmd.Value)
	case Unknown:
		fmt.Printf("Unknown command: %s\n", cmd.Value)
	}
}
