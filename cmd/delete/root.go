package delete

import (
	"github.com/spf13/cobra"
)

type commandList []*cobra.Command

func (p *commandList) AddCommand(cmd *cobra.Command) {
	list := *p
	list = append(list, cmd)
	*p = list
}

var commands commandList

// GetCommands returns all commands in slice
func GetCommands() []*cobra.Command {
	return commands
}
