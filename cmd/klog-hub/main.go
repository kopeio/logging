package main

import (
	"crypto/tls"
	"crypto/x509"
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
	gitRepo = "https://kope.io/klogs"
)

type Options struct {
	Loghub                         loghub.Options
	GrpcPublicTlsCert              string
	GrpcPublicTlsKey               string
	GrpcPublicAuthenticationMethod string
	KubernetesAuthenticationUrl    string
	KubernetesAuthenticationCA     string
}

func main() {
	flags := pflag.NewFlagSet("", pflag.ExitOnError)

	var options Options

	options.Loghub.SetDefaults()

	options.GrpcPublicAuthenticationMethod = "kubernetes"

	// TODO: Should there be a kubernetes service in kube-system?
	options.KubernetesAuthenticationUrl = "https://kubernetes.default/api"
	options.KubernetesAuthenticationCA = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	flags.StringVar(&options.Loghub.LogGRPC.Listen, "grpc-public-listen", options.Loghub.LogGRPC.Listen, "Address on which to listen for public request")
	flags.StringVar(&options.Loghub.MeshGRPC.Listen, "grpc-mesh-listen", options.Loghub.MeshGRPC.Listen, "Address on which to listen for internal requests")
	flags.StringVar(&options.GrpcPublicTlsCert, "grpc-public-tls-cert", options.GrpcPublicTlsCert, "Path to TLS certificate")
	flags.StringVar(&options.GrpcPublicTlsKey, "grpc-public-tls-key", options.GrpcPublicTlsKey, "Path to TLS private key")

	flags.StringVar(&options.GrpcPublicAuthenticationMethod, "grpc-public-authentication", options.GrpcPublicAuthenticationMethod, "Authentication method to use")
	flags.StringVar(&options.KubernetesAuthenticationUrl, "kubernetes-auth-url", options.KubernetesAuthenticationUrl, "Kubernetes authentication URL to use")

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

	err = Run(&options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func Run(options *Options) error {
	var err error
	if options.GrpcPublicTlsCert != "" {
		options.Loghub.LogGRPC.TLSCert, err = ioutil.ReadFile(options.GrpcPublicTlsCert)
		if err != nil {
			return fmt.Errorf("error reading file %q: %v", options.GrpcPublicTlsCert, err)
		}
	}

	if options.GrpcPublicTlsKey != "" {
		options.Loghub.LogGRPC.TLSKey, err = ioutil.ReadFile(options.GrpcPublicTlsKey)
		if err != nil {
			return fmt.Errorf("error reading file %q: %v", options.GrpcPublicTlsKey, err)
		}
	}

	if options.GrpcPublicAuthenticationMethod == "kubernetes" {
		tlsConfig := &tls.Config{}
		if options.KubernetesAuthenticationCA != "" {
			rootCAs := x509.NewCertPool()

			pemData, err := ioutil.ReadFile(options.KubernetesAuthenticationCA)
			if err != nil {
				return fmt.Errorf("error reading file %q: %v", options.KubernetesAuthenticationCA, err)
			}
			if !rootCAs.AppendCertsFromPEM(pemData) {
				return fmt.Errorf("unable to parse ca certificate file %q: %v", options.KubernetesAuthenticationCA, err)
			}
			tlsConfig.RootCAs = rootCAs
		}

		options.Loghub.LogGRPC.Authorizer = grpc.NewKubernetesAuthorizer(options.KubernetesAuthenticationUrl, tlsConfig)
	} else {
		// options.LogGRPC.Authorizer = grpc.NewTokenAuthorizer([]string{grpcPublicToken})
		return fmt.Errorf("unknown authentication method %q", options.GrpcPublicAuthenticationMethod)
	}

	err = loghub.ListenAndServe(&options.Loghub)
	if err != nil {
		return fmt.Errorf("error running server: %v", err)
	}

	return err
}
