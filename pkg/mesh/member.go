package mesh

import (
	"fmt"
	"github.com/golang/glog"
	"kope.io/klog/pkg/proto"
	"google.golang.org/grpc"
	"net/url"
	"sync"
	"time"
)

type Member struct {
	id       string
	hostInfo proto.HostInfo

	mutex      sync.Mutex
	logsClient proto.LogServerClient
}

func (h *Member) run() {
	for {
		err := h.runOnce()
		if err != nil {
			glog.Warningf("error polling host %q: %v", h.id, err)
		}
		time.Sleep(5 * time.Second)
	}
}

func (h *Member) update(request *proto.JoinMeshRequest) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.hostInfo.Url != request.HostInfo.Url {
		h.logsClient = nil
	}
	h.hostInfo = *request.HostInfo
}

func (h *Member) runOnce() error {
	return nil
}

func (h *Member) LogsClient() (proto.LogServerClient, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	client := h.logsClient

	if client == nil {
		u, err := url.Parse(h.hostInfo.Url)
		if err != nil {
			return nil, fmt.Errorf("invalid host url %q", h.hostInfo.Url)
		}

		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		conn, err := grpc.Dial(u.Host, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to mesh client: %v", err)
		}
		client = proto.NewLogServerClient(conn)
		h.logsClient = client
	}

	return client, nil
}

func (m *Member) Id() string {
	return m.id
}
