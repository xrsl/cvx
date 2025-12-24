// Package style provides consistent terminal styling for the cvx CLI.
// Inspired by Python's Typer library for rich help output.
package style

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// ANSI color codes
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Italic = "\033[3m"

	Red     = "\033[0;31m"
	Green   = "\033[0;32m"
	Yellow  = "\033[1;33m"
	Blue    = "\033[0;34m"
	Magenta = "\033[0;35m"
	Cyan    = "\033[0;36m"
	Gray    = "\033[90m"
)

// NoColor disables colors (for non-TTY or --no-color flag)
var NoColor = false

func init() {
	// Disable colors if CVX_NO_COLOR env is set
	if os.Getenv("CVX_NO_COLOR") != "" {
		NoColor = true
	}
	// Check if stdout is a TTY
	if fileInfo, err := os.Stdout.Stat(); err == nil {
		if (fileInfo.Mode() & os.ModeCharDevice) == 0 {
			NoColor = true
		}
	}
}

// C wraps text with color, respecting NoColor setting
func C(color, text string) string {
	if NoColor {
		return text
	}
	return color + text + Reset
}

// B makes text bold
func B(text string) string {
	if NoColor {
		return text
	}
	return Bold + text + Reset
}

// Success formats a success message
func Success(label string) string {
	return C(Green, label+":") + Reset + " "
}

// SetupHelp configures Typer-style help templates for a Cobra command
func SetupHelp(cmd *cobra.Command) {
	cobra.AddTemplateFunc("styleHeading", styleHeading)
	cobra.AddTemplateFunc("styleCommand", styleCommand)
	cobra.AddTemplateFunc("styleDefault", styleDefault)
	cobra.AddTemplateFunc("rpadStyled", rpadStyled)

	cmd.SetUsageTemplate(usageTemplate)
	cmd.SetHelpTemplate(helpTemplate)
}

func styleHeading(s string) string {
	if NoColor {
		return s
	}
	return Bold + Magenta + s + Reset
}

func styleCommand(s string) string {
	if NoColor {
		return s
	}
	return Cyan + s + Reset
}

func styleDefault(s string) string {
	if NoColor {
		return s
	}
	return Gray + s + Reset
}

func rpadStyled(s string, padding int) string {
	styled := styleCommand(s)
	// Add padding based on raw string length
	padLen := padding - len(s)
	if padLen > 0 {
		return styled + strings.Repeat(" ", padLen)
	}
	return styled
}

// usageTemplate is the Typer-style usage template
const usageTemplate = `{{ styleHeading "Usage:" }}
  {{ styleCommand .UseLine }}{{if .HasAvailableSubCommands}} [command]{{end}}
{{if .HasAvailableSubCommands}}
{{ styleHeading "Commands:" }}{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpadStyled .Name .NamePadding }}  {{.Short}}{{end}}{{end}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// helpTemplate is the Typer-style help template
const helpTemplate = `{{if .Long}}{{.Long}}

{{else if .Short}}{{.Short}}

{{end}}{{ styleHeading "Usage:" }}
  {{ styleCommand .UseLine }}{{if .HasAvailableSubCommands}} [command]{{end}}
{{if .HasExample}}
{{ styleHeading "Examples:" }}
{{.Example}}
{{end}}{{if .HasAvailableSubCommands}}
{{ styleHeading "Commands:" }}{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpadStyled .Name .NamePadding }}  {{.Short}}{{end}}{{end}}
{{end}}{{if .HasAvailableLocalFlags}}
{{ styleHeading "Options:" }}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableInheritedFlags}}
{{ styleHeading "Global Options:" }}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableSubCommands}}
Use "{{.CommandPath}} [command] --help" for more information about a command.
{{end}}`
