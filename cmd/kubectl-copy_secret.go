package main

import (
	"fmt"
	"github.com/fleetingclarity/kubectl-copy_secret/pkg/cmd"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-copy_secret", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewCmdCopySecret(genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}, initClient())
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func initClient() *kubernetes.Clientset {
	home, _ := os.UserHomeDir()
	kubeConfigPath := filepath.Join(home, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		// we may be inside the cluster, try that approach and panic if not
		originalErr := err.Error()
		config, err = rest.InClusterConfig()
		if err != nil {
			fmt.Printf("\noriginal error: %s\nlatest error: %s\n", originalErr, err)
			panic(err)
		}
	}
	return kubernetes.NewForConfigOrDie(config)
}
