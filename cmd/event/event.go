package installevent

import (
	"context"
	"helm.sh/helm/v4/pkg/cli"
	"io"

	"helm.sh/helm/v4/pkg/action"
)

type event interface {
	FinishInstall(settings *cli.EnvSettings, cfg *action.Configuration, name string) error
	WaitTestCaseFinish(settings *cli.EnvSettings, ctx context.Context, out io.Writer) error
	CheckUninstall(settings *cli.EnvSettings, name string, cfg *action.Configuration, out io.Writer) error
	QueryRunningPod(settings *cli.EnvSettings, ctx context.Context, cfg *action.Configuration, out io.Writer)
}
