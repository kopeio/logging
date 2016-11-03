package mesh

import (
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"kope.io/klog/pkg/grpc"
	"kope.io/klog/pkg/proto"
	"sync"
)

type Server struct {
	grpc *grpc.GRPCServer

	mutex   sync.Mutex
	members map[string]*Member
}

var _ proto.MeshServiceServer = &Server{}

func NewServer(options *grpc.GRPCOptions) (*Server, error) {
	grpcServer, err := grpc.NewGrpcServer(options)
	if err != nil {
		return nil, err
	}

	s := &Server{
		grpc:    grpcServer,
		members: make(map[string]*Member),
	}

	proto.RegisterMeshServiceServer(grpcServer.Server, s)

	return s, nil
}

func (s *Server) ListenAndServe() error {
	return s.grpc.ListenAndServe()
}

func (s *Server) JoinMesh(context context.Context, request *proto.JoinMeshRequest) (*proto.JoinMeshResponse, error) {
	glog.Infof("JoinMesh %s", request)

	if request.HostInfo == nil {
		return nil, fmt.Errorf("HostInfo not set")
	}

	id := request.HostInfo.Id
	if id == "" {
		return nil, fmt.Errorf("HostInfo.Id not set")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	h := s.members[id]
	isNew := false
	if h == nil {
		h = &Member{id: id}
		s.members[id] = h
		isNew = true
	}
	h.update(request)

	if isNew {
		go h.run()
	}
	response := &proto.JoinMeshResponse{}
	return response, nil
}

// Hosts returns a snapshot of the hosts
func (s *Server) Members() []*Member {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	members := make([]*Member, 0, len(s.members))
	for _, m := range s.members {
		members = append(members, m)
	}
	return members
}
