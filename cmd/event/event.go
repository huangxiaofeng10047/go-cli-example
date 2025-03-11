package installevent

import (
	"context"
	"helm.sh/helm/v4/pkg/action"
	"io"
)

type event interface {
	FinishInstall(cfg *action.Configuration, name string) error
	WaitTestCaseFinish(ctx context.Context, out io.Writer) error
}
