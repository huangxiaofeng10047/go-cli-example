package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"helm.sh/helm/v4/cmd/helm/require"
)

const completionDesc = `
Generate autocompletion scripts for Helm for the specified shell.
`
const bashCompDesc = `
Generate the autocompletion script for Helm for the bash shell.

To load completions in your current shell session:

    source <(helm completion bash)

To load completions for every new session, execute once:
- Linux:

      helm completion bash > /etc/bash_completion.d/helm

- MacOS:

      helm completion bash > /usr/local/etc/bash_completion.d/helm
`

const zshCompDesc = `
Generate the autocompletion script for Helm for the zsh shell.

To load completions in your current shell session:

    source <(helm completion zsh)

To load completions for every new session, execute once:

    helm completion zsh > "${fpath[1]}/_helm"
`

const fishCompDesc = `
Generate the autocompletion script for Helm for the fish shell.

To load completions in your current shell session:

    helm completion fish | source

To load completions for every new session, execute once:

    helm completion fish > ~/.config/fish/completions/helm.fish

You will need to start a new shell for this setup to take effect.
`

const powershellCompDesc = `
Generate the autocompletion script for powershell.

To load completions in your current shell session:
PS C:\> helm completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.
`

const (
	noDescFlagName = "no-descriptions"
	noDescFlagText = "disable completion descriptions"
)

var disableCompDescriptions bool

func newCompletionCmd(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "generate autocompletion scripts for the specified shell",
		Long:  completionDesc,
		Args:  require.NoArgs,
	}

	bash := &cobra.Command{
		Use:               "bash",
		Short:             "generate autocompletion script for bash",
		Long:              bashCompDesc,
		Args:              require.NoArgs,
		ValidArgsFunction: noMoreArgsCompFunc,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCompletionBash(out, cmd)
		},
	}
	bash.Flags().BoolVar(&disableCompDescriptions, noDescFlagName, false, noDescFlagText)

	zsh := &cobra.Command{
		Use:               "zsh",
		Short:             "generate autocompletion script for zsh",
		Long:              zshCompDesc,
		Args:              require.NoArgs,
		ValidArgsFunction: noMoreArgsCompFunc,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCompletionZsh(out, cmd)
		},
	}
	zsh.Flags().BoolVar(&disableCompDescriptions, noDescFlagName, false, noDescFlagText)

	fish := &cobra.Command{
		Use:               "fish",
		Short:             "generate autocompletion script for fish",
		Long:              fishCompDesc,
		Args:              require.NoArgs,
		ValidArgsFunction: noMoreArgsCompFunc,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCompletionFish(out, cmd)
		},
	}
	fish.Flags().BoolVar(&disableCompDescriptions, noDescFlagName, false, noDescFlagText)

	powershell := &cobra.Command{
		Use:               "powershell",
		Short:             "generate autocompletion script for powershell",
		Long:              powershellCompDesc,
		Args:              require.NoArgs,
		ValidArgsFunction: noMoreArgsCompFunc,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCompletionPowershell(out, cmd)
		},
	}
	powershell.Flags().BoolVar(&disableCompDescriptions, noDescFlagName, false, noDescFlagText)

	cmd.AddCommand(bash, zsh, fish, powershell)

	return cmd
}

func runCompletionBash(out io.Writer, cmd *cobra.Command) error {
	err := cmd.Root().GenBashCompletionV2(out, !disableCompDescriptions)

	// In case the user renamed the helm binary (e.g., to be able to run
	// both helm2 and helm3), we hook the new binary name to the completion function
	if binary := filepath.Base(os.Args[0]); binary != "helm" {
		renamedBinaryHook := `
# Hook the command used to generate the completion script
# to the helm completion function to handle the case where
# the user renamed the helm binary
if [[ $(type -t compopt) = "builtin" ]]; then
    complete -o default -F __start_helm %[1]s
else
    complete -o default -o nospace -F __start_helm %[1]s
fi
`
		fmt.Fprintf(out, renamedBinaryHook, binary)
	}

	return err
}

func runCompletionZsh(out io.Writer, cmd *cobra.Command) error {
	var err error
	if disableCompDescriptions {
		err = cmd.Root().GenZshCompletionNoDesc(out)
	} else {
		err = cmd.Root().GenZshCompletion(out)
	}

	// In case the user renamed the helm binary (e.g., to be able to run
	// both helm2 and helm3), we hook the new binary name to the completion function
	if binary := filepath.Base(os.Args[0]); binary != "helm" {
		renamedBinaryHook := `
# Hook the command used to generate the completion script
# to the helm completion function to handle the case where
# the user renamed the helm binary
compdef _helm %[1]s
`
		fmt.Fprintf(out, renamedBinaryHook, binary)
	}

	// Cobra doesn't source zsh completion file, explicitly doing it here
	fmt.Fprintf(out, "compdef _helm helm")

	return err
}

func runCompletionFish(out io.Writer, cmd *cobra.Command) error {
	return cmd.Root().GenFishCompletion(out, !disableCompDescriptions)
}

func runCompletionPowershell(out io.Writer, cmd *cobra.Command) error {
	if disableCompDescriptions {
		return cmd.Root().GenPowerShellCompletion(out)
	}
	return cmd.Root().GenPowerShellCompletionWithDesc(out)
}

// noMoreArgsCompFunc deactivates file completion when doing argument shell completion.
// It also provides some ActiveHelp to indicate no more arguments are accepted.
func noMoreArgsCompFunc(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	return noMoreArgsComp()
}

// noMoreArgsComp deactivates file completion when doing argument shell completion.
// It also provides some ActiveHelp to indicate no more arguments are accepted.
func noMoreArgsComp() ([]string, cobra.ShellCompDirective) {
	activeHelpMsg := "This command does not take any more arguments (but may accept flags)."
	return cobra.AppendActiveHelp(nil, activeHelpMsg), cobra.ShellCompDirectiveNoFileComp
}
