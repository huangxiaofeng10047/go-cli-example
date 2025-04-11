/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	installevent "github.com/huangxiaofeng10047/go-cli-example/cmd/event"
	tlsutil "github.com/huangxiaofeng10047/go-cli-example/cmd/helm"
	"helm.sh/helm/v4/pkg/cli"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/registry"
	"helm.sh/helm/v4/pkg/repo"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	//err := rootCmd.Execute()
	//if err != nil {
	//	os.Exit(1)
	//}
}

var (
	testConfig = &installevent.TestConfig{
		Schema: "http",
		Host:   "127.0.0.1",
		Port:   8080,
	}
	globalEvent *installevent.InstallEvent
)
var globalUsage = `The Kubernetes package manager
Common actions for Helm:

- helm search:    search for charts
- helm pull:      download a chart to your local directory to view
- helm install:   upload the chart to Kubernetes
- helm list:      list releases of charts

Environment variables:

| Name                               | Description                                                                                                |
|------------------------------------|------------------------------------------------------------------------------------------------------------|
| $HELM_CACHE_HOME                   | set an alternative location for storing cached files.                                                      |
| $HELM_CONFIG_HOME                  | set an alternative location for storing Helm configuration.                                                |
| $HELM_DATA_HOME                    | set an alternative location for storing Helm data.                                                         |
| $HELM_DEBUG                        | indicate whether or not Helm is running in Debug mode                                                      |
| $HELM_DRIVER                       | set the backend storage driver. Values are: configmap, secret, memory, sql.                                |
| $HELM_DRIVER_SQL_CONNECTION_STRING | set the connection string the SQL storage driver should use.                                               |
| $HELM_MAX_HISTORY                  | set the maximum number of helm release history.                                                            |
| $HELM_NAMESPACE                    | set the namespace used for the helm operations.                                                            |
| $HELM_NO_PLUGINS                   | disable plugins. Set HELM_NO_PLUGINS=1 to disable plugins.                                                 |
| $HELM_PLUGINS                      | set the path to the plugins directory                                                                      |
| $HELM_REGISTRY_CONFIG              | set the path to the registry config file.                                                                  |
| $HELM_REPOSITORY_CACHE             | set the path to the repository cache directory                                                             |
| $HELM_REPOSITORY_CONFIG            | set the path to the repositories file.                                                                     |
| $KUBECONFIG                        | set an alternative Kubernetes configuration file (default "~/.kube/config")                                |
| $HELM_KUBEAPISERVER                | set the Kubernetes API Server Endpoint for authentication                                                  |
| $HELM_KUBECAFILE                   | set the Kubernetes certificate authority file.                                                             |
| $HELM_KUBEASGROUPS                 | set the Groups to use for impersonation using a comma-separated list.                                      |
| $HELM_KUBEASUSER                   | set the Username to impersonate for the operation.                                                         |
| $HELM_KUBECONTEXT                  | set the name of the kubeconfig context.                                                                    |
| $HELM_KUBETOKEN                    | set the Bearer KubeToken used for authentication.                                                          |
| $HELM_KUBEINSECURE_SKIP_TLS_VERIFY | indicate if the Kubernetes API server's certificate validation should be skipped (insecure)                |
| $HELM_KUBETLS_SERVER_NAME          | set the server name used to validate the Kubernetes API server certificate                                 |
| $HELM_BURST_LIMIT                  | set the default burst limit in the case the server contains many CRDs (default 100, -1 to disable)         |
| $HELM_QPS                          | set the Queries Per Second in cases where a high number of calls exceed the option for higher burst values |

Helm stores cache, configuration, and data based on the following configuration order:

- If a HELM_*_HOME environment variable is set, it will be used
- Otherwise, on systems supporting the XDG base directory specification, the XDG variables will be used
- When no other location is set a default location will be used based on the operating system

By default, the default directories depend on the Operating System. The defaults are listed below:

| Operating System | Cache Path                | Configuration Path             | Data Path               |
|------------------|---------------------------|--------------------------------|-------------------------|
| Linux            | $HOME/.cache/helm         | $HOME/.config/helm             | $HOME/.local/share/helm |
| macOS            | $HOME/Library/Caches/helm | $HOME/Library/Preferences/helm | $HOME/Library/helm      |
| Windows          | %TEMP%\helm               | %APPDATA%\helm                 | %APPDATA%\helm          |
`

func NewRootCmd(settings *cli.EnvSettings, actionConfig *action.Configuration, out io.Writer, args []string, debug action.DebugLog) (*cobra.Command, error) {
	globalEvent = installevent.NewInstallEvent(testConfig)
	cmd := &cobra.Command{
		Use:          "helm",
		Short:        "The Helm package manager for Kubernetes.",
		Long:         globalUsage,
		SilenceUsage: true,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			globalEvent = installevent.NewInstallEvent(testConfig)
			if err := startProfiling(); err != nil {
				log.Printf("Warning: Failed to start profiling: %v", err)
			}
		},
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			if err := stopProfiling(); err != nil {
				log.Printf("Warning: Failed to stop profiling: %v", err)
			}
		},
	}
	flags := cmd.PersistentFlags()
	flags.StringVar(&testConfig.Schema, "test-case-schema", "http", "测试用例协议")
	flags.StringVar(&testConfig.Host, "test-case-host", "127.0.0.1", "测试用例主机地址")
	flags.IntVar(&testConfig.Port, "test-case-port", 8080, "测试用例端口")
	settings.AddFlags(flags)
	addKlogFlags(flags)

	// Setup shell completion for the namespace flag
	err := cmd.RegisterFlagCompletionFunc("namespace", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		if client, err := actionConfig.KubernetesClientSet(); err == nil {
			// Choose a long enough timeout that the user notices something is not working
			// but short enough that the user is not made to wait very long
			to := int64(3)
			cobra.CompDebugln(fmt.Sprintf("About to call kube client for namespaces with timeout of: %d", to), settings.Debug)

			nsNames := []string{}
			if namespaces, err := client.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{TimeoutSeconds: &to}); err == nil {
				for _, ns := range namespaces.Items {
					nsNames = append(nsNames, ns.Name)
				}
				return nsNames, cobra.ShellCompDirectiveNoFileComp
			}
		}
		return nil, cobra.ShellCompDirectiveDefault
	})

	if err != nil {
		log.Fatal(err)
	}

	// Setup shell completion for the kube-context flag
	err = cmd.RegisterFlagCompletionFunc("kube-context", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		cobra.CompDebugln("About to get the different kube-contexts", settings.Debug)

		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		if len(settings.KubeConfig) > 0 {
			loadingRules = &clientcmd.ClientConfigLoadingRules{ExplicitPath: settings.KubeConfig}
		}
		if config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules,
			&clientcmd.ConfigOverrides{}).RawConfig(); err == nil {
			comps := []string{}
			for name, context := range config.Contexts {
				comps = append(comps, fmt.Sprintf("%s\t%s", name, context.Cluster))
			}
			return comps, cobra.ShellCompDirectiveNoFileComp
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	})

	if err != nil {
		log.Fatal(err)
	}

	// We can safely ignore any errors that flags.Parse encounters since
	// those errors will be caught later during the call to cmd.Execution.
	// This call is required to gather configuration information prior to
	// execution.
	flags.ParseErrorsWhitelist.UnknownFlags = true
	flags.Parse(args)

	registryClient, err := newDefaultRegistryClient(settings, false, "", "")
	if err != nil {
		return nil, err
	}
	actionConfig.RegistryClient = registryClient

	// Add subcommands

	// 初始化全局 event
	//globalEvent = installevent.NewInstallEvent(testConfig)
	cmd.AddCommand(
		// chart commands
		newDependencyCmd(settings, actionConfig, out),
		newInstallCmd(settings, actionConfig, out, debug),
		newListCmd(settings, actionConfig, out, debug),
		newTemplateCmd(settings, actionConfig, out, debug),
	)
	// 使用 PersistentFlags 而不是 Flags

	//cmd.AddCommand(
	//	newRegistryCmd(actionConfig, out),
	//	newPushCmd(actionConfig, out),
	//)

	// Find and add plugins
	loadPlugins(settings, cmd, out)

	// Check for expired repositories
	checkForExpiredRepos(settings.RepositoryConfig)

	return cmd, nil
}
func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.go-cli-example.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// GetGlobalEvent 返回全局事件实例
func GetGlobalEvent() *installevent.InstallEvent {
	return globalEvent
}
func newRegistryClient(
	settings *cli.EnvSettings, certFile, keyFile, caFile string, insecureSkipTLSverify, plainHTTP bool, username, password string,
) (*registry.Client, error) {
	if certFile != "" && keyFile != "" || caFile != "" || insecureSkipTLSverify {
		registryClient, err := newRegistryClientWithTLS(settings, certFile, keyFile, caFile, insecureSkipTLSverify, username, password)
		if err != nil {
			return nil, err
		}
		return registryClient, nil
	}
	registryClient, err := newDefaultRegistryClient(settings, plainHTTP, username, password)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}
func newRegistryClientWithTLS(
	settings *cli.EnvSettings, certFile, keyFile, caFile string, insecureSkipTLSverify bool, username, password string,
) (*registry.Client, error) {
	tlsConf, err := tlsutil.NewTLSConfig(
		tlsutil.WithInsecureSkipVerify(insecureSkipTLSverify),
		tlsutil.WithCertKeyPairFiles(certFile, keyFile),
		tlsutil.WithCAFile(caFile),
	)

	if err != nil {
		return nil, fmt.Errorf("can't create TLS config for client: %w", err)
	}

	// Create a new registry client
	registryClient, err := registry.NewClient(
		registry.ClientOptDebug(settings.Debug),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stderr),
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
		registry.ClientOptHTTPClient(&http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConf,
			},
		}),
		registry.ClientOptBasicAuth(username, password),
	)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}
func newDefaultRegistryClient(settings *cli.EnvSettings, plainHTTP bool, username, password string) (*registry.Client, error) {
	opts := []registry.ClientOption{
		registry.ClientOptDebug(settings.Debug),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stderr),
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
		registry.ClientOptBasicAuth(username, password),
	}
	if plainHTTP {
		opts = append(opts, registry.ClientOptPlainHTTP())
	}

	// Create a new registry client
	registryClient, err := registry.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return registryClient, nil
}
func checkForExpiredRepos(repofile string) {

	expiredRepos := []struct {
		name string
		old  string
		new  string
	}{
		{
			name: "stable",
			old:  "kubernetes-charts.storage.googleapis.com",
			new:  "https://charts.helm.sh/stable",
		},
		{
			name: "incubator",
			old:  "kubernetes-charts-incubator.storage.googleapis.com",
			new:  "https://charts.helm.sh/incubator",
		},
	}

	// parse repo file.
	// Ignore the error because it is okay for a repo file to be unparsable at this
	// stage. Later checks will trap the error and respond accordingly.
	repoFile, err := repo.LoadFile(repofile)
	if err != nil {
		return
	}

	for _, exp := range expiredRepos {
		r := repoFile.Get(exp.name)
		if r == nil {
			return
		}

		if url := r.URL; strings.Contains(url, exp.old) {
			fmt.Fprintf(
				os.Stderr,
				"WARNING: %q is deprecated for %q and will be deleted Nov. 13, 2020.\nWARNING: You should switch to %q via:\nWARNING: helm repo add %q %q --force-update\n",
				exp.old,
				exp.name,
				exp.new,
				exp.name,
				exp.new,
			)
		}
	}

}
