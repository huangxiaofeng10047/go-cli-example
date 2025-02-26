package cmd

import (
	"helm.sh/helm/v4/pkg/cli"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"helm.sh/helm/v4/cmd/helm/require"
)

var repoHelm = `
This command consists of multiple subcommands to interact with chart repositories.

It can be used to add, remove, list, and index chart repositories.
`

func newRepoCmd(settings *cli.EnvSettings, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo add|remove|list|index|update [ARGS]",
		Short: "add, list, remove, update, and index chart repositories",
		Long:  repoHelm,
		Args:  require.NoArgs,
	}

	//cmd.AddCommand(newRepoAddCmd(out))
	cmd.AddCommand(newRepoListCmd(settings, out))
	//cmd.AddCommand(newRepoRemoveCmd(out))
	//cmd.AddCommand(newRepoIndexCmd(out))
	//cmd.AddCommand(newRepoUpdateCmd(out))

	return cmd
}

func isNotExist(err error) bool {
	return os.IsNotExist(errors.Cause(err))
}
