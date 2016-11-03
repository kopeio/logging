package main

import (
	"github.com/spf13/cobra"
	"io"
	"kope.io/klogs/pkg/client"
)

func NewCmdSearch(factory client.Factory, out io.Writer) *cobra.Command {
	options := &client.SearchOptions{}
	options.Output = client.OutputFormatDescribe
	cmd := &cobra.Command{
		Use:     "search",
		Aliases: []string{"s"},
		Short:   "search",
		Run: func(cmd *cobra.Command, args []string) {
			err := client.RunSearch(factory, out, args, options)
			if err != nil {
				exitWithError(err)
			}
		},
	}

	cmd.PersistentFlags().StringVarP(&options.Output, "ouptut", "o", options.Output, "Output format: raw, describe")

	return cmd
}
