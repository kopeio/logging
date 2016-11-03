package main

import (
	goflag "flag"
	"github.com/golang/glog"
	"kope.io/klog/pkg/client"
	"github.com/spf13/cobra"
	"io"
)

const DefaultServerUrl = "http://127.0.0.1:7777"

func NewRootCommand(out io.Writer) *cobra.Command {
	factory := &client.DefaultFactory{
		Server: DefaultServerUrl,
	}

	cmd := &cobra.Command{
		Use:   "klogs",
		Short: "klogs is kubernetes logs",
	}

	// Really just to force the import
	glog.Flush()
	cmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)

	cmd.PersistentFlags().StringVar(&factory.Server, "server", factory.Server, "Server to query")

	// create subcommands
	cmd.AddCommand(NewCmdStreams(factory, out))
	cmd.AddCommand(NewCmdSearch(factory, out))
	return cmd
}
