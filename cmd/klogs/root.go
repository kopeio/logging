package main

import (
	goflag "flag"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"io"
	"kope.io/klogs/pkg/client"
)

const DefaultServerUrl = "http://127.0.0.1:7777"

func NewRootCommand(out io.Writer) (*cobra.Command, error) {
	factory := &client.DefaultFactory{
		Server: DefaultServerUrl,
	}

	err := factory.LoadConfigurationFiles()
	if err != nil {
		return nil, err
	}

	cmd := &cobra.Command{
		Use:   "klogs",
		Short: "klogs is kubernetes logs",
	}

	// Really just to force the import
	glog.Flush()
	cmd.PersistentFlags().AddGoFlagSet(goflag.CommandLine)

	cmd.PersistentFlags().StringVar(&factory.Server, "server", factory.Server, "Server to query")
	cmd.PersistentFlags().StringVar(&factory.Token, "token", factory.Token, "Token to use to authenticate to the server")
	cmd.PersistentFlags().StringVarP(&factory.Username, "user", "u", factory.Username, "Username to use to authenticate to the server")
	cmd.PersistentFlags().VarP(newPasswordValue(factory.Password, &factory.Password), "password", "p", "Password to use to authenticate to the server")

	// create subcommands
	cmd.AddCommand(NewCmdStreams(factory, out))
	cmd.AddCommand(NewCmdSearch(factory, out))

	return cmd, nil
}
