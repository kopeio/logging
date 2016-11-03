package client

import (
	"kope.io/klog/pkg/grpc"
	"kope.io/klog/pkg/proto"
)

type Factory interface {
	LogServerClient() (proto.LogServerClient, error)
}

type DefaultFactory struct {
	Server string
}

var _ Factory = &DefaultFactory{}

func (f *DefaultFactory) LogServerClient() (proto.LogServerClient, error) {
	options := &grpc.GRPCClientOptions{
		Server: f.Server,
	}
	conn, err := grpc.NewGRPCClient(options)
	if err != nil {
		return nil, err
	}
	client := proto.NewLogServerClient(conn)
	return client, nil
}
