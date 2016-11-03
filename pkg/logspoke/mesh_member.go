package logspoke

import (
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"kope.io/klogs/pkg/proto"
	"net/url"
	"time"
)

type MeshMember struct {
	server *url.URL
	id     string
	listen *url.URL
}

func newMeshMember(options *Options) (*MeshMember, error) {
	if options.JoinHub == "" {
		return nil, fmt.Errorf("JoinHub not set")
	}

	if options.NodeName == "" {
		return nil, fmt.Errorf("NodeName not set")
	}

	serverUrl, err := url.Parse(options.JoinHub)
	if err != nil {
		return nil, fmt.Errorf("Invalid JoinHub url %q", options.JoinHub)
	}

	listenUrl, err := url.Parse(options.Listen)
	if err != nil {
		return nil, fmt.Errorf("Invalid listen url %q", options.Listen)
	}

	m := &MeshMember{
		id:     options.NodeName,
		server: serverUrl,
		listen: listenUrl,
	}
	return m, nil
}

func (m *MeshMember) Run() error {
	for {
		err := func() error {
			var opts []grpc.DialOption
			//if *tls {
			//	var sn string
			//	if *serverHostOverride != "" {
			//		sn = *serverHostOverride
			//	}
			//	var creds credentials.TransportCredentials
			//	if *caFile != "" {
			//		var err error
			//		creds, err = credentials.NewClientTLSFromFile(*caFile, sn)
			//		if err != nil {
			//			grpclog.Fatalf("Failed to create TLS credentials %v", err)
			//		}
			//	} else {
			//		creds = credentials.NewClientTLSFromCert(nil, sn)
			//	}
			//	opts = append(opts, grpc.WithTransportCredentials(creds))
			//} else {
			opts = append(opts, grpc.WithInsecure())
			//}
			conn, err := grpc.Dial(m.server.Host, opts...)
			if err != nil {
				return fmt.Errorf("failed to connect to mesh server: %v", err)
			}
			defer conn.Close()

			client := proto.NewMeshServiceClient(conn)
			if err != nil {
				return fmt.Errorf("error building mesh client: %v", err)
			}

			for {
				ctx := context.Background()
				request := &proto.JoinMeshRequest{
					HostInfo: &proto.HostInfo{
						Id:  m.id,
						Url: m.listen.Scheme + "://" + m.listen.Host,
					},
				}
				response, err := client.JoinMesh(ctx, request)
				if err != nil {
					return fmt.Errorf("unexpected response to mesh join: %v", err)
				}
				glog.V(2).Infof("mesh join response %s", response)
				time.Sleep(10 * time.Second)
			}

			return nil
		}()
		if err != nil {
			glog.Warningf("error joining mesh: %v", err)
		}
		time.Sleep(10 * time.Second)
	}
}
