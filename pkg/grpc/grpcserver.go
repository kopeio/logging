package grpc

import (
	"crypto/tls"
	"fmt"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"net/url"
)

type GRPCOptions struct {
	Listen  string
	TLSCert []byte
	TLSKey  []byte
}

type GRPCServer struct {
	listen url.URL
	Server *grpc.Server
}

func NewGrpcServer(options *GRPCOptions) (*GRPCServer, error) {
	u, err := url.Parse(options.Listen)
	if err != nil {
		return nil, fmt.Errorf("Invalid listen address %q", options.Listen)
	}

	g := &GRPCServer{
		listen: *u,
	}
	var opts []grpc.ServerOption
	if u.Scheme == "http" {
		// No options needed
	} else if u.Scheme == "https" {
		if options.TLSCert == nil {
			return nil, fmt.Errorf("https selected, but tls-cert not provided")
		}
		if options.TLSKey == nil {
			return nil, fmt.Errorf("https selected, but tls-key not provided")
		}
		cert, err := tls.X509KeyPair(options.TLSCert, options.TLSKey)
		if err != nil {
			return nil, err
		}

		creds := credentials.NewServerTLSFromCert(&cert)
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	} else {
		return nil, fmt.Errorf("scheme not recognized: %q", u.Scheme)
	}
	g.Server = grpc.NewServer(opts...)

	return g, nil
}

func (g *GRPCServer) ListenAndServe() error {
	glog.Infof("Listening on %s", g.listen)

	lis, err := net.Listen("tcp", g.listen.Host)
	if err != nil {
		return fmt.Errorf("Failed to listen on %q: %v", g.listen, err)
	}
	defer lis.Close()
	return g.Server.Serve(lis)
}
