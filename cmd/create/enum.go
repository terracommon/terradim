package create

import (
	"fmt"

	"github.com/spf13/cobra"
)

// enumCmd represents the create/enum command
var enumCmd = &cobra.Command{
	Use:   "enum",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("create/enum called")
	},
}

func init() {
	commands.AddCommand(enumCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// create/enumCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// create/enumCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
