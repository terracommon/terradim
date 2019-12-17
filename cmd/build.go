package cmd

import (
	"fmt"

	"github.com/imburbank/terradim/model"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build terradim directory to live layout",
	Long: `Build a terradim model from the src directory and write
	with resolved configs to the dst directory. For example:
...
WARNING: This command will replace the contents of the dst directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		src := viper.GetString("src")
		dst := viper.GetString("dst")
		fmt.Printf("Building from %s to %s\n", src, dst)

		t, buildConfig := buildModel(src, dst)
		if err := writeToFile(t, buildConfig); err != nil {
			panic(err)
		}
		fmt.Println("Build Complete")
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// buildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// buildCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	buildCmd.Flags().StringP("src", "s", "terraform/terradim", "Path terradim input")
	viper.BindPFlag("src", buildCmd.Flags().Lookup("src"))

	buildCmd.Flags().StringP("dst", "d", "terraform/live", "Path to build output")
	viper.BindPFlag("dst", buildCmd.Flags().Lookup("dst"))

	buildCmd.Flags().BoolP("verbose", "v", false, "Verbose output to stdout")
	viper.BindPFlag("verbose", buildCmd.Flags().Lookup("verbose"))
}

func buildModel(src, dst string) (*model.Tree, *model.BuildConfig) {
	if src[:2] == "./" {
		src = src[2:]
	}
	t, buildConfig := model.Create(src, dst)
	return t, buildConfig
}

func writeToFile(t *model.Tree, config *model.BuildConfig) error {
	err := model.Write(t, config)
	return err
}
