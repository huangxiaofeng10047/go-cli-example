package cmd

import (
	"helm.sh/helm/v4/pkg/cli"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"helm.sh/helm/v4/cmd/helm/require"
	"helm.sh/helm/v4/pkg/action"
)

const dependencyDesc = `
Manage the dependencies of a chart.

Helm charts store their dependencies in 'charts/'. For chart developers, it is
often easier to manage dependencies in 'Chart.yaml' which declares all
dependencies.

The dependency commands operate on that file, making it easy to synchronize
between the desired dependencies and the actual dependencies stored in the
'charts/' directory.

For example, this Chart.yaml declares two dependencies:

    # Chart.yaml
    dependencies:
    - name: nginx
      version: "1.2.3"
      repository: "https://example.com/charts"
    - name: memcached
      version: "3.2.1"
      repository: "https://another.example.com/charts"


The 'name' should be the name of a chart, where that name must match the name
in that chart's 'Chart.yaml' file.

The 'version' field should contain a semantic version or version range.

The 'repository' URL should point to a Chart Repository. Helm expects that by
appending '/index.yaml' to the URL, it should be able to retrieve the chart
repository's index. Note: 'repository' can be an alias. The alias must start
with 'alias:' or '@'.

Starting from 2.2.0, repository can be defined as the path to the directory of
the dependency charts stored locally. The path should start with a prefix of
"file://". For example,

    # Chart.yaml
    dependencies:
    - name: nginx
      version: "1.2.3"
      repository: "file://../dependency_chart/nginx"

If the dependency chart is retrieved locally, it is not required to have the
repository added to helm by "helm add repo". Version matching is also supported
for this case.
`

const dependencyListDesc = `
List all of the dependencies declared in a chart.

This can take chart archives and chart directories as input. It will not alter
the contents of a chart.

This will produce an error if the chart cannot be loaded.
`

func newDependencyCmd(settings *cli.EnvSettings, cfg *action.Configuration, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dependency update|build|list",
		Aliases: []string{"dep", "dependencies"},
		Short:   "manage a chart's dependencies",
		Long:    dependencyDesc,
		Args:    require.NoArgs,
	}

	cmd.AddCommand(newDependencyListCmd(out))
	cmd.AddCommand(newDependencyUpdateCmd(settings, cfg, out))
	cmd.AddCommand(newDependencyBuildCmd(settings, out))

	return cmd
}

func newDependencyListCmd(out io.Writer) *cobra.Command {
	client := action.NewDependency()
	cmd := &cobra.Command{
		Use:     "list CHART",
		Aliases: []string{"ls"},
		Short:   "list the dependencies for the given chart",
		Long:    dependencyListDesc,
		Args:    require.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			chartpath := "."
			if len(args) > 0 {
				chartpath = filepath.Clean(args[0])
			}
			return client.List(chartpath, out)
		},
	}

	f := cmd.Flags()

	f.UintVar(&client.ColumnWidth, "max-col-width", 80, "maximum column width for output table")
	return cmd
}

func addDependencySubcommandFlags(f *pflag.FlagSet, client *action.Dependency) {
	f.BoolVar(&client.Verify, "verify", false, "verify the packages against signatures")
	f.StringVar(&client.Keyring, "keyring", defaultKeyring(), "keyring containing public keys")
	f.BoolVar(&client.SkipRefresh, "skip-refresh", false, "do not refresh the local repository cache")
	f.StringVar(&client.Username, "username", "", "chart repository username where to locate the requested chart")
	f.StringVar(&client.Password, "password", "", "chart repository password where to locate the requested chart")
	f.StringVar(&client.CertFile, "cert-file", "", "identify HTTPS client using this SSL certificate file")
	f.StringVar(&client.KeyFile, "key-file", "", "identify HTTPS client using this SSL key file")
	f.BoolVar(&client.InsecureSkipTLSverify, "insecure-skip-tls-verify", false, "skip tls certificate checks for the chart download")
	f.BoolVar(&client.PlainHTTP, "plain-http", false, "use insecure HTTP connections for the chart download")
	f.StringVar(&client.CaFile, "ca-file", "", "verify certificates of HTTPS-enabled servers using this CA bundle")
}
