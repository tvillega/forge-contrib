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

// ExecuteCommand prints information based on the parsed command type and action details.
func ExecuteCommand(cmd ParsedCommand, username string, issueNumber int, repoFullName string) {
	var actionDesc string
	switch cmd.Type {
	case AddLabel:
		actionDesc = fmt.Sprintf("Add label %s to #%d", cmd.Value, issueNumber)
	case RmLabel:
		actionDesc = fmt.Sprintf("Remove label %s from #%d", cmd.Value, issueNumber)
	case AddMilestone:
		actionDesc = fmt.Sprintf("Add milestone %s to #%d", cmd.Value, issueNumber)
	case RmMilestone:
		actionDesc = fmt.Sprintf("Remove milestone from #%d", issueNumber)
	case SelfAssign:
		actionDesc = fmt.Sprintf("Self-assign to #%d (%s)", issueNumber, cmd.Value)
	case SelfUnassign:
		actionDesc = fmt.Sprintf("Self-unassign from #%d (%s)", issueNumber, cmd.Value)
	case Unknown:
		actionDesc = fmt.Sprintf("Unknown command: %s", cmd.Value)
	}

	if actionDesc != "" {
		fmt.Printf("[%s]: %s in %s\n", username, actionDesc, repoFullName)
	}
}
