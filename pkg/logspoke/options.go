package logspoke

import (
	"fmt"
	"github.com/golang/glog"
	"kope.io/klog/pkg/archive"
	"kope.io/klog/pkg/archive/s3archive"
	"io/ioutil"
	"net/url"
	"strings"
)

type Options struct {
	PodDir       string
	ContainerDir string
	ArchiveSink  string
	Listen       string
	JoinHub      string
	NodeName     string
}

func (o *Options) SetDefaults() {
	o.PodDir = "/var/lib/kubelet/pods"
	o.ContainerDir = "/var/lib/docker/containers"
	o.Listen = "http://:7777"
	o.NodeName = "@/etc/hostname"
}

type LogShipper struct {
	scraper    *Scraper
	logServer  *LogServer
	meshMember *MeshMember
}

func NewLogShipper(options *Options) (*LogShipper, error) {
	l := &LogShipper{}

	if strings.HasPrefix(options.NodeName, "@") {
		f := options.NodeName[1:]
		nodeName, err := ioutil.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("error reading node name from %q: %v", f, err)
		}
		options.NodeName = strings.TrimSpace(string(nodeName))
		glog.Infof("Read hostname from %q: %q", f, options.NodeName)
	}

	var archiveSink archive.Sink
	if options.ArchiveSink != "" {
		u, err := url.Parse(options.ArchiveSink)
		if err != nil {
			return nil, fmt.Errorf("invalid ArchiveSink %q: %q", options.ArchiveSink, err)
		}

		if u.Scheme == "s3" {
			archiveSink, err = s3archive.NewSink(u)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unknown scheme in ArchivePath %q", options.ArchiveSink)
		}
	}
	nodeState := newNodeState(archiveSink)

	logServer, err := newLogServer(options, nodeState)
	if err != nil {
		return nil, err
	}
	l.logServer = logServer

	scraper, err := newScraper(options, nodeState)
	if err != nil {
		return nil, err
	}
	l.scraper = scraper

	if options.JoinHub != "" {
		l.meshMember, err = newMeshMember(options)
		if err != nil {
			return nil, err
		}
	}
	return l, nil
}

func (l *LogShipper) Run() error {
	go l.scraper.Run()

	if l.meshMember != nil {
		go l.meshMember.Run()
	}
	return l.logServer.Run()
}
