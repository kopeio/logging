package logspoke

import (
	"github.com/golang/glog"
	"time"
)

type Scraper struct {
	pods       *PodsDirectory
	containers *ContainersDirectory
}

func newScraper(options *Options, nodeState *NodeState) (*Scraper, error) {
	scraper := &Scraper{}

	pods, err := NewPodsDirectory(options.PodDir, nodeState)
	if err != nil {
		return nil, err
	}
	scraper.pods = pods

	containers, err := NewContainerLogsDirectory(options.ContainerDir, nodeState)
	if err != nil {
		return nil, err
	}
	scraper.containers = containers

	return scraper, nil
}

func (s *Scraper) Run() error {
	for {
		if err := s.pods.Scan(); err != nil {
			glog.Warningf("error scanning pods directory: %v", err)
		}

		if err := s.containers.Scan(); err != nil {
			glog.Warningf("error scanning containers directory: %v", err)
		}

		time.Sleep(time.Minute)
	}
}
