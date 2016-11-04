package grpc

import (
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net/url"
	"strings"
)

const MetadataKeyToken = "token"

const MetadataKeyUsername = "user"
const MetadataKeyPassword = "password"

type GRPCClientOptions struct {
	Server string

	Token string

	Username string
	Password string
}

type tokenCreds struct {
	Token string
}

func (c *tokenCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		MetadataKeyToken: c.Token,
	}, nil
}

func (c *tokenCreds) RequireTransportSecurity() bool {
	return true
}

type basicAuthCreds struct {
	Username string
	Password string
}

func (c *basicAuthCreds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		MetadataKeyUsername: c.Username,
		MetadataKeyPassword: c.Password,
	}, nil
}

func (c *basicAuthCreds) RequireTransportSecurity() bool {
	return true
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
		// TODO: Unclear if we need to set this.  Feels prudent!
		sn := u.Host
		colonIndex := strings.Index(sn, ":")
		if colonIndex != -1 {
			sn = sn[:colonIndex]
		}

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

	if options.Token != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(&tokenCreds{Token: options.Token}))
	} else if options.Username != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(&basicAuthCreds{Username: options.Username, Password: options.Password}))
	}

	conn, err := grpc.Dial(u.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server %q: %v", u.Host, err)
	}
	return conn, nil
}
