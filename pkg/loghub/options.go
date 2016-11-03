package loghub

import (
	"github.com/golang/glog"
	"kope.io/klog/pkg/grpc"
	"kope.io/klog/pkg/mesh"
)

type Options struct {
	LogGRPC  grpc.GRPCOptions
	MeshGRPC grpc.GRPCOptions
}

func (o *Options) SetDefaults() {
	o.LogGRPC.Listen = "https://:7777"
	o.MeshGRPC.Listen = "http://:7878"
}

func ListenAndServe(options *Options) error {
	m, err := mesh.NewServer(&options.MeshGRPC)
	if err != nil {
		return err
	}

	logServer, err := newLogServer(&options.LogGRPC, m)
	if err != nil {
		return err
	}

	go func() {
		if err := m.ListenAndServe(); err != nil {
			// TODO: Futures?
			glog.Fatalf("error starting mesh: %v", err)
		}
	}()

	return logServer.ListenAndServe()
}
