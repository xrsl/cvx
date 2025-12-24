package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags
var Version = "dev"

func getVersion() string {
	if Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return Version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print cvx version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cvx %s\n", getVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
