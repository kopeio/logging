package logspoke

import (
	"fmt"
	"github.com/golang/glog"
	"io"
	"kope.io/klogs/pkg/proto"
	"os"
	"path"
	"time"
)

type PodsDirectory struct {
	basedir    string
	idlePeriod time.Duration

	state *NodeState
}

func NewPodsDirectory(basedir string, state *NodeState) (*PodsDirectory, error) {
	d := &PodsDirectory{
		basedir:    basedir,
		idlePeriod: idlePeriod,
		state:      state,
	}
	return d, nil
}

func (d *PodsDirectory) Scan() error {
	return d.scanPodsDir(d.basedir)
}

func (d *PodsDirectory) scanPodsDir(basepath string) error {
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

		glog.V(4).Infof("Found pod: %q", p)
		err = d.scanPodDirectory(p, name)
		if err != nil {
			return err
		}
	}

	d.state.CleanupPodLogs(names)

	return nil
}

func (d *PodsDirectory) scanPodDirectory(basepath string, podID string) error {
	p := path.Join(basepath, "volumes/kubernetes.io~empty-dir/logs")
	stat, err := os.Lstat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("error doing stat on %q: %v", p, err)
	}
	if !stat.IsDir() {
		return nil
	}

	podState := d.state.GetPodState(podID)

	glog.V(4).Infof("Found pod logs mount: %q", p)

	fileMap := make(map[string]struct{})
	err = d.scanLogsTree(p, podState, "", fileMap)
	if err != nil {
		return err
	}

	glog.Warningf("TODO: Remove files not in fileMap")

	return nil
}

func (d *PodsDirectory) scanLogsTree(basepath string, podState *PodState, relativePath string, fileMap map[string]struct{}) error {
	f, err := os.OpenFile(basepath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("error opening %q: %v", basepath, err)
	}
	defer f.Close()

	for {
		names, err := f.Readdirnames(512)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading directory %q: %v", basepath, err)
		}

		for _, name := range names {
			p := path.Join(basepath, name)
			// TOOD: Use dirent to find out if dirs or not?
			stat, err := os.Lstat(p)
			if err != nil {
				return fmt.Errorf("error doing lstat on %q: %v", p, err)
			}

			if stat.IsDir() {
				err = d.scanLogsTree(p, podState, path.Join(relativePath, name), fileMap)
				if err != nil {
					return err
				}
			} else {
				f := path.Join(relativePath, name)
				fileMap[f] = struct{}{}
				fields := &proto.Fields{}
				podState.foundFile(p, path.Join(relativePath, name), stat, fields)
			}
		}

	}
	return nil
}
