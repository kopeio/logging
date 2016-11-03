package loghub

import (
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"io"
	"kope.io/klogs/pkg/grpc"
	"kope.io/klogs/pkg/mesh"
	"kope.io/klogs/pkg/proto"
	"sync"
)

type LogServer struct {
	grpcServer *grpc.GRPCServer
	mesh       *mesh.Server
}

var _ proto.LogServerServer = &LogServer{}

func newLogServer(options *grpc.GRPCOptions, mesh *mesh.Server) (*LogServer, error) {
	grpcServer, err := grpc.NewGrpcServer(options)
	if err != nil {
		return nil, err
	}

	s := &LogServer{
		grpcServer: grpcServer,
		mesh:       mesh,
	}

	proto.RegisterLogServerServer(grpcServer.Server, s)

	return s, nil
}

func (s *LogServer) ListenAndServe() error {
	return s.grpcServer.ListenAndServe()
}

func (s *LogServer) GetStreams(request *proto.GetStreamsRequest, out proto.LogServer_GetStreamsServer) error {
	ctx := out.Context()

	members := s.mesh.Members()
	for _, member := range members {
		// TODO: Run in parallel?

		client, err := member.LogsClient()
		if err != nil {
			// TODO: retries / toleration
			return fmt.Errorf("error fetching client for %q: %v", member.Id(), err)
		}

		stream, err := client.GetStreams(ctx, request)
		if err != nil {
			// TODO: retries / toleration
			return fmt.Errorf("error querying member %q: %v", member.Id(), err)
		}

		for {
			in, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				// TODO: retries / toleration
				return fmt.Errorf("error reading from member %q: %v", member.Id(), err)
			}
			err = out.Send(in)
			if err != nil {
				return fmt.Errorf("error sending results: %v", err)
			}
		}
	}

	return nil
}

func (s *LogServer) Search(request *proto.SearchRequest, out proto.LogServer_SearchServer) error {
	// TODO: Copy-pasted
	ctx := out.Context()

	members := s.mesh.Members()

	glog.Warningf("member filtering not implemented")

	var sendMutex sync.Mutex
	var wg sync.WaitGroup
	ops := make([]*DistributedOp, len(members))
	wg.Add(len(members))
	for i, member := range members {
		search := &DistributedOp{
			ctx:    ctx,
			member: member,
		}
		ops[i] = search

		go func(search *DistributedOp) {
			search.err = search.Search(&sendMutex, request, out)
			wg.Done()
		}(search)
	}

	wg.Wait()

	for _, op := range ops {
		if op.err != nil {
			return fmt.Errorf("error from member %q: %v", op.member.Id(), op.err)
		}
	}

	return nil
}

type DistributedOp struct {
	ctx    context.Context
	member *mesh.Member
	err    error
}

func (s *DistributedOp) Search(sendMutex *sync.Mutex, request *proto.SearchRequest, out proto.LogServer_SearchServer) error {
	client, err := s.member.LogsClient()
	if err != nil {
		// TODO: retries / toleration
		return fmt.Errorf("error fetching client: %v", err)
	}

	stream, err := client.Search(s.ctx, request)
	if err != nil {
		// TODO: retries / toleration
		return fmt.Errorf("error querying member: %v", err)
	}

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// TODO: retries / toleration
			return fmt.Errorf("error reading result: %v", err)
		}
		sendMutex.Lock()
		err = out.Send(in)
		sendMutex.Unlock()
		if err != nil {
			return fmt.Errorf("error sending results: %v", err)
		}
	}

	return nil
}
