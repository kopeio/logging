package client

import (
	"kope.io/klogs/pkg/grpc"
	"kope.io/klogs/pkg/proto"
)

type Factory interface {
	LogServerClient() (proto.LogServerClient, error)
}

type DefaultFactory struct {
	Server string
	Token  string
}

var _ Factory = &DefaultFactory{}

func (f *DefaultFactory) LogServerClient() (proto.LogServerClient, error) {
	options := &grpc.GRPCClientOptions{
		Server: f.Server,
		Token:  f.Token,
	}
	conn, err := grpc.NewGRPCClient(options)
	if err != nil {
		return nil, err
	}
	client := proto.NewLogServerClient(conn)
	return client, nil
}
