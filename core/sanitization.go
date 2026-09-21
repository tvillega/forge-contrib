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
	if strings.HasPrefix(commandStr, addLabel) {
		return ParsedCommand{
			Type:  AddLabel,
			Value: strings.TrimPrefix(commandStr, addLabel),
		}
	} else if strings.HasPrefix(commandStr, rmLabel) {
		return ParsedCommand{
			Type:  RmLabel,
			Value: strings.TrimPrefix(commandStr, rmLabel),
		}
	} else if strings.HasPrefix(commandStr, addMilestone) {
		return ParsedCommand{
			Type:  AddMilestone,
			Value: strings.TrimPrefix(commandStr, addMilestone),
		}
	} else if strings.HasPrefix(commandStr, rmMilestone) {
		return ParsedCommand{
			Type:  RmMilestone,
			Value: strings.TrimPrefix(commandStr, rmMilestone),
		}
	} else if strings.HasPrefix(commandStr, selfAssign) {
		return ParsedCommand{
			Type:  SelfAssign,
			Value: strings.TrimPrefix(commandStr, selfAssign),
		}
	} else if strings.HasPrefix(commandStr, selfUnassign) {
		return ParsedCommand{
			Type:  SelfUnassign,
			Value: strings.TrimPrefix(commandStr, selfUnassign),
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
