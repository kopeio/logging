package logspoke

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"kope.io/klog/pkg/proto"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

type ContainersDirectory struct {
	containersDir string
	state         *NodeState
}

func NewContainerLogsDirectory(containersDir string, state *NodeState) (*ContainersDirectory, error) {
	d := &ContainersDirectory{
		containersDir: containersDir,
		state:         state,
	}
	return d, nil
}

func (d *ContainersDirectory) Scan() error {
	return d.scanContainersDir(d.containersDir)
}

func (d *ContainersDirectory) scanContainersDir(basepath string) error {
	f, err := os.OpenFile(basepath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("error opening %q: %v", basepath, err)
	}
	defer f.Close()

	names, err := f.Readdirnames(-1)
	if err != nil {
		return fmt.Errorf("error reading directory %q: %v", basepath, err)
	}

	for _, name := range names {
		p := path.Join(basepath, name)

		glog.V(4).Infof("Found container directory: %q", p)
		err = d.scanContainerDirectory(p, name)
		if err != nil {
			return err
		}
	}

	d.state.CleanupContainerLogs(names)

	return nil
}

type DockerConfigV2 struct {
	Config DockerConfigV2_Config
}

type DockerConfigV2_Config struct {
	Image  string
	Labels map[string]string
}

func tryReadConfig(containerID string, containerDir string) *proto.Fields {
	configPath := path.Join(containerDir, "config.v2.json")
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Ignore
			glog.V(4).Infof("No config.v2.json file in %q", containerDir)
		} else {
			glog.Warningf("error reading file %q: %v", configPath, err)
		}
		return nil
	}

	config := &DockerConfigV2{}
	err = json.Unmarshal(data, config)
	if err != nil {
		glog.Warningf("error parsing file %q: %v", configPath, err)
		return nil
	}

	fields := &proto.Fields{}
	fields.Fields = append(fields.Fields, &proto.Field{
		Key:   "container.id",
		Value: containerID,
	})

	for k, v := range config.Config.Labels {
		switch k {
		case "io.kubernetes.container.name":
			fields.Fields = append(fields.Fields, &proto.Field{
				Key:   "container.name",
				Value: v,
			})
		case "io.kubernetes.pod.name":
			fields.Fields = append(fields.Fields, &proto.Field{
				Key:   "pod.name",
				Value: v,
			})
		case "io.kubernetes.pod.namespace":
			fields.Fields = append(fields.Fields, &proto.Field{
				Key:   "pod.namespace",
				Value: v,
			})
		case "io.kubernetes.pod.uid":
			fields.Fields = append(fields.Fields, &proto.Field{
				Key:   "pod.uid",
				Value: v,
			})
		}
	}

	//"io.kubernetes.container.hash": "8053578f",
	//	"io.kubernetes.container.name": "kubedns",
	//	"io.kubernetes.container.ports": "[{\"name\":\"dns-local\",\"containerPort\":10053,\"protocol\":\"UDP\"},{\"name\":\"dns-tcp-local\",\"containerPort\":10053,\"protocol\":\"TCP\"}]",
	//	"io.kubernetes.container.restartCount": "0",
	//	"io.kubernetes.container.terminationMessagePath": "/dev/termination-log",
	//	"io.kubernetes.pod.name": "kube-dns-v20-90109312-80wgs",
	//	"io.kubernetes.pod.namespace": "kube-system",
	//	"io.kubernetes.pod.terminationGracePeriod": "30",
	//	"io.kubernetes.pod.uid": "854530ad-975a-11e6-b8af-06e5bea45582"

	//"io.kubernetes.container.hash": "d8dbe16c",
	//	"io.kubernetes.container.name": "POD",
	//	"io.kubernetes.container.restartCount": "0",
	//	"io.kubernetes.container.terminationMessagePath": "",
	//	"io.kubernetes.pod.name": "kube-dns-v20-90109312-xojr5",
	//	"io.kubernetes.pod.namespace": "kube-system",
	//	"io.kubernetes.pod.terminationGracePeriod": "30",
	//	"io.kubernetes.pod.uid": "86d89496-975a-11e6-b8af-06e5bea45582"

	return fields
}

func (d *ContainersDirectory) scanContainerDirectory(containerDir string, containerID string) error {
	containerState := d.state.GetContainerState(containerID)

	fields := tryReadConfig(containerID, containerDir)

	glog.V(4).Infof("Found container: %q", containerID)

	fileMap := make(map[string]struct{})

	f, err := os.OpenFile(containerDir, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("error opening %q: %v", containerDir, err)
	}
	defer f.Close()
	for {
		names, err := f.Readdirnames(512)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading directory %q: %v", containerDir, err)
		}

		for _, name := range names {
			if !strings.HasPrefix(name, containerID+"-json.log") {
				switch name {
				case "hostconfig.json", "config.v2.json", "resolv.conf", "resolv.conf.hash", "hosts", "shm", "hostname":
				// Ignore
				default:
					glog.Infof("Ignoring unknown file %q", name)
				}
				continue
			}

			p := path.Join(containerDir, name)
			glog.Infof("Found container log file %q", p)

			fileMap[name] = struct{}{}
			stat, err := os.Lstat(p)
			if err != nil {
				if !os.IsNotExist(err) {
					glog.Warningf("error doing lstat on file %q: %v", p, err)
				}
				continue
			}

			containerState.foundFile(p, name, stat, fields)
		}

		glog.Warningf("TODO: Remove files not in fileMap")
	}
	return nil
}
