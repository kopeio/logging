package main

import (
	"github.com/spf13/cobra"
	"io"
	"kope.io/klog/pkg/client"
)

func NewCmdStreams(factory client.Factory, out io.Writer) *cobra.Command {
	options := &client.ListStreamsOptions{}

	cmd := &cobra.Command{
		Use:   "streams",
		Short: "Streams",
		Run: func(cmd *cobra.Command, args []string) {
			err := client.RunListStreams(factory, out, options)
			if err != nil {
				exitWithError(err)
			}
		},
	}

	return cmd
}
