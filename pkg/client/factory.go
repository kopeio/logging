package client

import (
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"kope.io/klogs/pkg/grpc"
	"kope.io/klogs/pkg/proto"
	"os"
	"path/filepath"
	"strings"
)

type Factory interface {
	LogServerClient() (proto.LogServerClient, error)
}

type DefaultFactory struct {
	Server   string `json:"server,omitempty"`
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

var _ Factory = &DefaultFactory{}

func (f *DefaultFactory) LogServerClient() (proto.LogServerClient, error) {
	options := &grpc.GRPCClientOptions{
		Server:   f.Server,
		Token:    f.Token,
		Username: f.Username,
		Password: f.Password,
	}
	conn, err := grpc.NewGRPCClient(options)
	if err != nil {
		return nil, err
	}
	client := proto.NewLogServerClient(conn)
	return client, nil
}

func (f *DefaultFactory) LoadConfigurationFiles() error {
	var paths []string

	home := os.Getenv("HOME")
	if home != "" {
		paths = append(paths, filepath.Join(home, ".klogs", "config.yaml"))
	}

	for _, p := range paths {
		data, err := ioutil.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("error reading config file %q: %v", p, err)
		}

		s := strings.TrimSpace(string(data))
		if s == "" {
			continue
		}

		glog.V(2).Infof("Parsing config file %q", p)
		err = yaml.Unmarshal([]byte(s), f)
		if err != nil {
			return fmt.Errorf("error parsing config file %q: %v", p, err)
		}
		return nil
	}

	return nil
}
