/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	mycmd "github.com/huangxiaofeng10047/go-cli-example/cmd"
	"github.com/spf13/cobra"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/cli"
	kubefake "helm.sh/helm/v4/pkg/kube/fake"
	"helm.sh/helm/v4/pkg/release"
	"helm.sh/helm/v4/pkg/storage/driver"
	"io"
	"log"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
	"time"
)

var settings = cli.New()

func warning(format string, v ...interface{}) {
	format = fmt.Sprintf("WARNING: %s\n", format)
	fmt.Fprintf(os.Stderr, format, v...)
}
func debug(format string, v ...interface{}) {
	if settings.Debug {
		timeNow := time.Now().String()
		format = fmt.Sprintf("%s [debug] %s\n", timeNow, format)
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

// hookOutputWriter provides the writer for writing hook logs.
func hookOutputWriter(_, _, _ string) io.Writer {
	return log.Writer()
}
func main() {
	actionConfig := new(action.Configuration)
	cmd, err := mycmd.NewRootCmd(settings, actionConfig, os.Stdout, os.Args[1:], debug)
	if err != nil {
		warning("%+v", err)
		os.Exit(1)
	}
	// run when each command's execute method is called
	cobra.OnInitialize(func() {
		helmDriver := os.Getenv("HELM_DRIVER")
		if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), helmDriver, debug); err != nil {
			log.Fatal(err)
		}
		if helmDriver == "memory" {
			loadReleasesInMemory(actionConfig)
		}
		actionConfig.SetHookOutputFunc(hookOutputWriter)
	})

	if err := cmd.Execute(); err != nil {
		debug("%+v", err)
		switch e := err.(type) {
		case pluginError:
			os.Exit(e.code)
		default:
			os.Exit(1)
		}
	}
}

type pluginError struct {
	error
	code int
}

func loadReleasesInMemory(actionConfig *action.Configuration) {
	filePaths := strings.Split(os.Getenv("HELM_MEMORY_DRIVER_DATA"), ":")
	if len(filePaths) == 0 {
		return
	}

	store := actionConfig.Releases
	mem, ok := store.Driver.(*driver.Memory)
	if !ok {
		// For an unexpected reason we are not dealing with the memory storage driver.
		return
	}

	actionConfig.KubeClient = &kubefake.PrintingKubeClient{Out: io.Discard}

	for _, path := range filePaths {
		b, err := os.ReadFile(path)
		if err != nil {
			log.Fatal("Unable to read memory driver data", err)
		}

		releases := []*release.Release{}
		if err := yaml.Unmarshal(b, &releases); err != nil {
			log.Fatal("Unable to unmarshal memory driver data: ", err)
		}

		for _, rel := range releases {
			if err := store.Create(rel); err != nil {
				log.Fatal(err)
			}
		}
	}
	// Must reset namespace to the proper one
	mem.SetNamespace(settings.Namespace())
}
