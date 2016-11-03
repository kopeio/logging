package client

import (
	"fmt"
	"golang.org/x/net/context"
	"io"
	"kope.io/klog/pkg/proto"
)

type ListStreamsOptions struct {
}

func RunListStreams(f Factory, out io.Writer, o *ListStreamsOptions) error {
	request := &proto.GetStreamsRequest{}
	client, err := f.LogServerClient()
	if err != nil {
		return err
	}

	// TODO: What is the right context?
	ctx := context.Background()

	stream, err := client.GetStreams(ctx, request)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading from server: %v", err)
		}
		_, err = fmt.Fprintf(out, "%v\n", in)
		if err != nil {
			return fmt.Errorf("error writing results: %v", err)
		}
	}

	return nil
}
