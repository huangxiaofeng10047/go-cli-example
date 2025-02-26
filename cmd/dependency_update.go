package cmd

import (
	"fmt"
	"helm.sh/helm/v4/pkg/cli"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"

	"helm.sh/helm/v4/cmd/helm/require"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/downloader"
	"helm.sh/helm/v4/pkg/getter"
)

const dependencyUpDesc = `
Update the on-disk dependencies to mirror Chart.yaml.

This command verifies that the required charts, as expressed in 'Chart.yaml',
are present in 'charts/' and are at an acceptable version. It will pull down
the latest charts that satisfy the dependencies, and clean up old dependencies.

On successful update, this will generate a lock file that can be used to
rebuild the dependencies to an exact version.

Dependencies are not required to be represented in 'Chart.yaml'. For that
reason, an update command will not remove charts unless they are (a) present
in the Chart.yaml file, but (b) at the wrong version.
`

// newDependencyUpdateCmd creates a new dependency update command.
func newDependencyUpdateCmd(settings *cli.EnvSettings, _ *action.Configuration, out io.Writer) *cobra.Command {
	client := action.NewDependency()

	cmd := &cobra.Command{
		Use:     "update CHART",
		Aliases: []string{"up"},
		Short:   "update charts/ based on the contents of Chart.yaml",
		Long:    dependencyUpDesc,
		Args:    require.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			chartpath := "."
			if len(args) > 0 {
				chartpath = filepath.Clean(args[0])
			}
			registryClient, err := newRegistryClient(settings, client.CertFile, client.KeyFile, client.CaFile,
				client.InsecureSkipTLSverify, client.PlainHTTP, client.Username, client.Password)
			if err != nil {
				return fmt.Errorf("missing registry client: %w", err)
			}

			man := &downloader.Manager{
				Out:              out,
				ChartPath:        chartpath,
				Keyring:          client.Keyring,
				SkipUpdate:       client.SkipRefresh,
				Getters:          getter.All(settings),
				RegistryClient:   registryClient,
				RepositoryConfig: settings.RepositoryConfig,
				RepositoryCache:  settings.RepositoryCache,
				Debug:            settings.Debug,
			}
			if client.Verify {
				man.Verify = downloader.VerifyAlways
			}
			return man.Update()
		},
	}

	f := cmd.Flags()
	addDependencySubcommandFlags(f, client)

	return cmd
}
