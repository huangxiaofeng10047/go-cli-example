package cmd

import (
	"context"
	"fmt"
	installevent "github.com/huangxiaofeng10047/go-cli-example/cmd/event"
	"helm.sh/helm/v4/pkg/cli"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"                 // 添加这行导入
	_ "k8s.io/client-go/kubernetes/typed/core/v1" // 添加这行导入
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"helm.sh/helm/v4/cmd/helm/require"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart/v2/loader"
	"helm.sh/helm/v4/pkg/cli/output"
	"helm.sh/helm/v4/pkg/cli/values"
	"helm.sh/helm/v4/pkg/downloader"
	"helm.sh/helm/v4/pkg/getter"
	"helm.sh/helm/v4/pkg/release"
)

const sacleDesc = `
This command installs a chart archive.

    扩容或者缩容pulsar服务
    $ go-cli-example scale --replicase 0 -n pulsar-test

To see the list of chart repositories, use 'helm repo list'. To search for
charts in a repository, use 'helm search'.
`

func newScaleCmd(settings *cli.EnvSettings, cfg *action.Configuration, out io.Writer, debug action.DebugLog) *cobra.Command {
	client := action.NewInstall(cfg)
	valueOpts := &values.Options{}
	var outfmt output.Format
	var replicas int
	var namespace string

	cmd := &cobra.Command{
		Use:   "scale [NAME] [CHART]",
		Short: "scale statefulSet of the cluster",
		Long:  sacleDesc,
		Args:  require.MinimumNArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return compInstall(settings, args, toComplete, client)
		},
		RunE: func(_ *cobra.Command, args []string) error {
			event := installevent.NewInstallEvent(testConfig)
			ctx := context.Background()
			registryClient, err := newRegistryClient(settings, client.CertFile, client.KeyFile, client.CaFile,
				client.InsecureSkipTLSverify, client.PlainHTTP, client.Username, client.Password)
			if err != nil {
				return fmt.Errorf("missing registry client: %w", err)
			}
			client.SetRegistryClient(registryClient)

			if client.DryRunOption == "" {
				client.DryRunOption = "none"
			}

			// 处理缩容逻辑
			if replicas == 0 {
				if len(args) == 0 {
					return fmt.Errorf("请提供要匹配的 StatefulSet 前缀")
				}
				prefix := args[0]
				fmt.Printf("要匹配的前缀: %s\n", prefix)    // 添加调试信息
				fmt.Printf("目标命名空间: %s\n", namespace) // 添加调试信息

				config, err := cfg.RESTClientGetter.ToRESTConfig()
				if err != nil {
					return fmt.Errorf("failed to get REST config: %w", err)
				}
				clientset, err := kubernetes.NewForConfig(config)
				if err != nil {
					return fmt.Errorf("failed to create clientset: %w", err)
				}

				// 列出指定命名空间下的所有 StatefulSet
				statefulSetList, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
				if err != nil {
					return fmt.Errorf("failed to list statefulsets: %w", err)
				}

				var matchingStatefulSets []metav1.ObjectMeta
				// 手动过滤出符合前缀的 StatefulSet
				for _, ss := range statefulSetList.Items {
					if strings.HasPrefix(ss.Name, prefix+"-") {
						matchingStatefulSets = append(matchingStatefulSets, ss.ObjectMeta)
					}
				}

				fmt.Printf("找到 %d 个匹配的 StatefulSet\n", len(matchingStatefulSets)) // 添加调试信息

				patch := []byte(fmt.Sprintf(`{"spec": {"replicas": %d}}`, replicas))
				for _, ssMeta := range matchingStatefulSets {
					fmt.Printf("正在缩容 StatefulSet: %s\n", ssMeta.Name) // 添加调试信息
					_, err = clientset.AppsV1().StatefulSets(namespace).Patch(ctx, ssMeta.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
					if err != nil {
						return fmt.Errorf("failed to scale down statefulSet %s: %w", ssMeta.Name, err)
					}
					fmt.Fprintf(out, "Successfully scaled down %s to %d replicas in namespace %s\n", ssMeta.Name, replicas, namespace)
				}
				return nil
			}

			rel, err := runScale(settings, args, client, valueOpts, out, debug)
			if err != nil {
				return errors.Wrap(err, "scaled FAILED")
			}
			err = outfmt.Write(out, &statusPrinter{
				release:      rel,
				debug:        settings.Debug,
				showMetadata: false,
				hideNotes:    client.HideNotes,
			})
			if err != nil {
				return errors.Wrap(err, "scaled FAILED")
			}
			err = event.QueryRunningPod(settings, ctx, cfg, out)
			if err != nil {
				return err
			}
			taskID, err2 := event.FinishInstall(settings, cfg, args[0])
			if err2 != nil {
				return errors.Wrap(err, "scaled FAILED")
			}
			fmt.Fprintln(out, "Waiting for testcase finish...")

			err = event.WaitTestCaseFinish(settings, ctx, out, taskID)
			if err != nil {
				return err
			}
			return nil
		},
	}

	addScaleFlags(settings, cmd, cmd.Flags(), client, valueOpts)
	// 添加 --replicase 和 -n 标志
	cmd.Flags().IntVar(&replicas, "replicase", 0, "Number of replicas to scale the statefulSet to")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Namespace to scale the statefulSet in")
	// hide-secret is not available in all places the install flags are used so
	// it is added separately
	f := cmd.Flags()
	f.BoolVar(&client.HideSecret, "hide-secret", false, "hide Kubernetes Secrets when also using the --dry-run flag")
	bindOutputFlag(cmd, &outfmt)
	bindPostRenderFlag(cmd, &client.PostRenderer)

	return cmd
}

func addScaleFlags(settings *cli.EnvSettings, cmd *cobra.Command, f *pflag.FlagSet, client *action.Install, valueOpts *values.Options) {
	//f.BoolVar(&client.CreateNamespace, "create-namespace", false, "create the release namespace if not present")
	//// --dry-run options with expected outcome:
	//// - Not set means no dry run and server is contacted.
	//// - Set with no value, a value of client, or a value of true and the server is not contacted
	//// - Set with a value of false, none, or false and the server is contacted
	//// The true/false part is meant to reflect some legacy behavior while none is equal to "".
	//f.StringVar(&client.DryRunOption, "dry-run", "", "simulate an install. If --dry-run is set with no option being specified or as '--dry-run=client', it will not attempt cluster connections. Setting '--dry-run=server' allows attempting cluster connections.")
	//f.Lookup("dry-run").NoOptDefVal = "client"
	//f.BoolVar(&client.Force, "force", false, "force resource updates through a replacement strategy")
	//f.BoolVar(&client.DisableHooks, "no-hooks", false, "prevent hooks from running during install")
	//f.BoolVar(&client.Replace, "replace", false, "reuse the given name, only if that name is a deleted release which remains in the history. This is unsafe in production")
	//f.DurationVar(&client.Timeout, "timeout", 300*time.Second, "time to wait for any individual Kubernetes operation (like Jobs for hooks)")
	//f.BoolVar(&client.Wait, "wait", false, "if set, will wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment, StatefulSet, or ReplicaSet are in a ready state before marking the release as successful. It will wait for as long as --timeout")
	//f.BoolVar(&client.WaitForJobs, "wait-for-jobs", false, "if set and --wait enabled, will wait until all Jobs have been completed before marking the release as successful. It will wait for as long as --timeout")
	//f.BoolVarP(&client.GenerateName, "generate-name", "g", false, "generate the name (and omit the NAME parameter)")
	//f.StringVar(&client.NameTemplate, "name-template", "", "specify template used to name the release")
	//f.StringVar(&client.Description, "description", "", "add a custom description")
	//f.BoolVar(&client.Devel, "devel", false, "use development versions, too. Equivalent to version '>0.0.0-0'. If --version is set, this is ignored")
	//f.BoolVar(&client.DependencyUpdate, "dependency-update", false, "update dependencies if they are missing before installing the chart")
	//f.BoolVar(&client.DisableOpenAPIValidation, "disable-openapi-validation", false, "if set, the installation process will not validate rendered templates against the Kubernetes OpenAPI Schema")
	//f.BoolVar(&client.Atomic, "atomic", false, "if set, the installation process deletes the installation on failure. The --wait flag will be set automatically if --atomic is used")
	//f.BoolVar(&client.SkipCRDs, "skip-crds", false, "if set, no CRDs will be installed. By default, CRDs are installed if not already present")
	//f.BoolVar(&client.SubNotes, "render-subchart-notes", false, "if set, render subchart notes along with the parent")
	//f.BoolVar(&client.SkipSchemaValidation, "skip-schema-validation", false, "if set, disables JSON schema validation")
	//f.StringToStringVarP(&client.Labels, "labels", "l", nil, "Labels that would be added to release metadata. Should be divided by comma.")
	//f.BoolVar(&client.EnableDNS, "enable-dns", false, "enable DNS lookups when rendering templates")
	//f.BoolVar(&client.HideNotes, "hide-notes", false, "if set, do not show notes in install output. Does not affect presence in chart metadata")
	//f.BoolVar(&client.TakeOwnership, "take-ownership", false, "if set, install will ignore the check for helm annotations and take ownership of the existing resources")
	addValueOptionsFlags(f, valueOpts)
	addChartPathOptionsFlags(f, &client.ChartPathOptions)

	err := cmd.RegisterFlagCompletionFunc("version", func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		requiredArgs := 2
		if client.GenerateName {
			requiredArgs = 1
		}
		if len(args) != requiredArgs {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return compVersionFlag(settings, args[requiredArgs-1], toComplete)
	})
	if err != nil {
		log.Fatal(err)
	}
}

func runScale(settings *cli.EnvSettings, args []string, client *action.Install, valueOpts *values.Options, out io.Writer, debug action.DebugLog) (*release.Release, error) {
	debug("Original chart version: %q", client.Version)
	if client.Version == "" && client.Devel {
		debug("setting version to >0.0.0-0")
		client.Version = ">0.0.0-0"
	}

	name, chart, err := client.NameAndChart(args)
	if err != nil {
		return nil, err
	}
	client.ReleaseName = name

	cp, err := client.ChartPathOptions.LocateChart(chart, settings)
	if err != nil {
		return nil, err
	}

	debug("CHART PATH: %s\n", cp)

	p := getter.All(settings)
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return nil, err
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return nil, err
	}

	if err := checkIfInstallable(chartRequested); err != nil {
		return nil, err
	}

	if chartRequested.Metadata.Deprecated {
		debug("This chart is deprecated")
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			err = errors.Wrap(err, "An error occurred while checking for chart dependencies. You may need to run `helm dependency build` to fetch missing dependencies")
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              out,
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
					Debug:            settings.Debug,
					RegistryClient:   client.GetRegistryClient(),
				}
				if err := man.Update(); err != nil {
					return nil, err
				}
				// Reload the chart with the updated Chart.lock file.
				if chartRequested, err = loader.Load(cp); err != nil {
					return nil, errors.Wrap(err, "failed reloading chart after repo update")
				}
			} else {
				return nil, err
			}
		}
	}

	client.Namespace = settings.Namespace()

	// Validate DryRunOption member is one of the allowed values
	if err := validateDryRunOptionFlag(client.DryRunOption); err != nil {
		return nil, err
	}

	// Create context and prepare the handle of SIGTERM
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	cSignal := make(chan os.Signal, 2)
	signal.Notify(cSignal, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-cSignal
		fmt.Fprintf(out, "Release %s has been cancelled.\n", args[0])
		cancel()
	}()

	return client.RunWithContext(ctx, chartRequested, vals)
}
