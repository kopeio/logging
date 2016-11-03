package grpc

import (
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net/url"
)

type GRPCClientOptions struct {
	Server string
}

func NewGRPCClient(options *GRPCClientOptions) (*grpc.ClientConn, error) {
	u, err := url.Parse(options.Server)
	if err != nil {
		return nil, fmt.Errorf("Invalid server url %q", options.Server)
	}

	var opts []grpc.DialOption
	if u.Scheme == "http" {
		opts = append(opts, grpc.WithInsecure())
	} else if u.Scheme == "https" {
		var sn string
		//if *serverHostOverride != "" {
		//	sn = *serverHostOverride
		//}
		var creds credentials.TransportCredentials
		//if *caFile != "" {
		//	var err error
		//	creds, err = credentials.NewClientTLSFromFile(*caFile, sn)
		//	if err != nil {
		//		grpclog.Fatalf("Failed to create TLS credentials %v", err)
		//	}
		//} else {
		//	creds = credentials.NewClientTLSFromCert(nil, sn)
		//}
		creds = credentials.NewClientTLSFromCert(nil, sn)
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		return nil, fmt.Errorf("unknown scheme %q", u.Scheme)
	}
	conn, err := grpc.Dial(u.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server %q: %v", u.Host, err)
	}
	return conn, nil
}
