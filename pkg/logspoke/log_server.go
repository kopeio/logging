package logspoke

import (
	"fmt"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"kope.io/klogs/pkg/proto"
	"net"
	"net/url"
)

type LogServer struct {
	listen    string
	logServer proto.LogServerServer
}

func newLogServer(options *Options, logServer proto.LogServerServer) (*LogServer, error) {
	if options.Listen == "" {
		return nil, fmt.Errorf("MeshListen not set")
	}

	m := &LogServer{
		listen:    options.Listen,
		logServer: logServer,
	}
	return m, nil
}

func (m *LogServer) Run() error {
	u, err := url.Parse(m.listen)
	if err != nil {
		return fmt.Errorf("Invalid listen url %q", m.listen)
	}

	glog.Infof("Serving GRPC on %s", m.listen)

	lis, err := net.Listen("tcp", u.Host)
	if err != nil {
		return fmt.Errorf("Failed to listen on %q: %v", m.listen, err)
	}
	defer lis.Close()
	var opts []grpc.ServerOption
	//if *tls {
	//	creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
	//	if err != nil {
	//		grpclog.Fatalf("Failed to generate credentials %v", err)
	//	}
	//	opts = []grpc.ServerOption{grpc.Creds(creds)}
	//}
	grpcServer := grpc.NewServer(opts...)
	proto.RegisterLogServerServer(grpcServer, m.logServer)
	err = grpcServer.Serve(lis)
	if err != nil {
		return fmt.Errorf("error running grpc server: %v", err)
	}
	return nil
}
