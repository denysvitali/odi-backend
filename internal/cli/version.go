package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/denysvitali/odi-backend/internal/ui"
)

// Version information - these can be set at build time via ldflags
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version, commit hash, build date, and Go version.`,
	Run:   runVersion,
}

func runVersion(cmd *cobra.Command, args []string) {
	ui.PrintHeader("ODI - Open Document Indexer")
	fmt.Printf("  Version:    %s\n", Version)
	fmt.Printf("  Commit:     %s\n", Commit)
	fmt.Printf("  Built:      %s\n", BuildDate)
	fmt.Printf("  Go version: %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
