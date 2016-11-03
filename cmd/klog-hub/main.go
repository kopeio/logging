package main

import (
	goflag "flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"io/ioutil"
	"kope.io/klogs/pkg/grpc"
	"kope.io/klogs/pkg/loghub"
	"os"
	"strings"
)

var (
	// value overwritten during build. This can be used to resolve issues.
	version = "0.1"
	gitRepo = "https://kope.io/klog"
)

func main() {
	flags := pflag.NewFlagSet("", pflag.ExitOnError)

	options := loghub.Options{}
	options.SetDefaults()

	var grpcPublicTlsCert string
	var grpcPublicTlsKey string
	var grpcPublicToken string

	flags.StringVar(&options.LogGRPC.Listen, "grpc-public-listen", options.LogGRPC.Listen, "Address on which to listen for public request")
	flags.StringVar(&options.MeshGRPC.Listen, "grpc-mesh-listen", options.MeshGRPC.Listen, "Address on which to listen for internal requests")
	flags.StringVar(&grpcPublicTlsCert, "grpc-public-tls-cert", grpcPublicTlsCert, "Path to TLS certificate")
	flags.StringVar(&grpcPublicTlsKey, "grpc-public-tls-key", grpcPublicTlsKey, "Path to TLS private key")
	flags.StringVar(&grpcPublicToken, "grpc-public-token", grpcPublicToken, "Token required (use something better in production!)")

	// Trick to avoid 'logging before flag.Parse' warning
	goflag.CommandLine.Parse([]string{})

	goflag.Set("logtostderr", "true")

	flags.AddGoFlagSet(goflag.CommandLine)
	//clientConfig := kubectl_util.DefaultClientConfig(flags)

	args := os.Args

	flagsPath := "/config/flags.yaml"
	_, err := os.Lstat(flagsPath)
	if err == nil {
		flagsFile, err := ioutil.ReadFile(flagsPath)
		if err != nil {
			glog.Fatalf("error reading %q: %v", flagsPath, err)
		}

		for _, line := range strings.Split(string(flagsFile), "\n") {
			line = strings.TrimSpace(line)
			args = append(args, line)
		}
	} else if !os.IsNotExist(err) {
		glog.Infof("Cannot read %q: %v", flagsPath, err)
	}

	flags.Parse(args)

	glog.Infof("loghub - build: %v - %v", gitRepo, version)

	if grpcPublicTlsCert != "" {
		options.LogGRPC.TLSCert, err = ioutil.ReadFile(grpcPublicTlsCert)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading file %q: %v", grpcPublicTlsCert, err)
		}
	}

	if grpcPublicTlsKey != "" {
		options.LogGRPC.TLSKey, err = ioutil.ReadFile(grpcPublicTlsKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading file %q: %v", grpcPublicTlsKey, err)
		}
	}

	if grpcPublicToken != "" {
		options.LogGRPC.Authorizer = grpc.NewTokenAuthorizer([]string{grpcPublicToken})
	}

	err = loghub.ListenAndServe(&options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
		os.Exit(1)
	}
}
