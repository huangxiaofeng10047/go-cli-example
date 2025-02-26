package cmd

import (
	"fmt"
	"helm.sh/helm/v4/pkg/cli"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"

	"helm.sh/helm/v4/cmd/helm/require"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/downloader"
	"helm.sh/helm/v4/pkg/getter"
)

const dependencyBuildDesc = `
Build out the charts/ directory from the Chart.lock file.

Build is used to reconstruct a chart's dependencies to the state specified in
the lock file. This will not re-negotiate dependencies, as 'helm dependency update'
does.

If no lock file is found, 'helm dependency build' will mirror the behavior
of 'helm dependency update'.
`

func newDependencyBuildCmd(settings *cli.EnvSettings, out io.Writer) *cobra.Command {
	client := action.NewDependency()

	cmd := &cobra.Command{
		Use:   "build CHART",
		Short: "rebuild the charts/ directory based on the Chart.lock file",
		Long:  dependencyBuildDesc,
		Args:  require.MaximumNArgs(1),
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
				man.Verify = downloader.VerifyIfPossible
			}
			err = man.Build()
			if e, ok := err.(downloader.ErrRepoNotFound); ok {
				return fmt.Errorf("%s. Please add the missing repos via 'helm repo add'", e.Error())
			}
			return err
		},
	}

	f := cmd.Flags()
	addDependencySubcommandFlags(f, client)

	return cmd
}

// defaultKeyring returns the expanded path to the default keyring.
func defaultKeyring() string {
	if v, ok := os.LookupEnv("GNUPGHOME"); ok {
		return filepath.Join(v, "pubring.gpg")
	}
	return filepath.Join(homedir.HomeDir(), ".gnupg", "pubring.gpg")
}
