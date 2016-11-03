package main

import (
	goflag "flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	"io/ioutil"
	"kope.io/klogs/pkg/logspoke"
	"net"
	"os"
	"strings"
)

var (
	// value overwritten during build. This can be used to resolve issues.
	version = "0.1"
	gitRepo = "https://kope.io/klog"
)

func main() {
	flags := pflag.NewFlagSet("", pflag.ExitOnError)

	options := logspoke.Options{}
	options.SetDefaults()

	podIP, err := findPodIP()
	if err != nil {
		fmt.Fprintf(os.Stderr, "r: %v\n", err)
		os.Exit(1)
	}
	options.Listen = "http://" + podIP.String() + ":7777"

	flags.StringVar(&options.PodDir, "pod-dir", options.PodDir, "Directory where pods files are stored")
	flags.StringVar(&options.ContainerDir, "container-dir", options.ContainerDir, "Directory where container files are stored")
	flags.StringVar(&options.ArchiveSink, "archive", options.ArchiveSink, "Destination to upload archived files")
	flags.StringVar(&options.JoinHub, "hub", options.JoinHub, "Hub server to register with")
	flags.StringVar(&options.Listen, "listen", options.Listen, "Address on which to listen")
	flags.StringVar(&options.NodeName, "nodename", options.NodeName, "Node name, or @path to load from path")

	// Trick to avoid 'logging before flag.Parse' warning
	goflag.CommandLine.Parse([]string{})

	goflag.Set("logtostderr", "true")

	flags.AddGoFlagSet(goflag.CommandLine)
	//clientConfig := kubectl_util.DefaultClientConfig(flags)

	args := os.Args

	flagsPath := "/config/flags.yaml"
	_, err = os.Lstat(flagsPath)
	if err == nil {
		flagsFile, err := ioutil.ReadFile(flagsPath)
		if err != nil {
			glog.Fatalf("error reading %q: %v", flagsPath, err)
		}

		for _, line := range strings.Split(string(flagsFile), "\n") {
			line = strings.TrimSpace(line)
			args = append(args, line)
		}
	} else if !os.IsNotExist(err) {
		glog.Infof("Cannot read %q: %v", flagsPath, err)
	}

	flags.Parse(args)

	glog.Infof("klog-spoke - build: %v - %v", gitRepo, version)

	s, err := logspoke.NewLogShipper(&options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
		os.Exit(1)
	}

	err = s.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "unexpected error: %v\n", err)
		os.Exit(1)
	}
}

func findPodIP() (net.IP, error) {
	var ips []net.IP

	networkInterfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("error querying interfaces to determine pod ip: %v", err)
	}

	for i := range networkInterfaces {
		networkInterface := &networkInterfaces[i]
		flags := networkInterface.Flags
		name := networkInterface.Name

		if (flags & net.FlagLoopback) != 0 {
			glog.V(2).Infof("Ignoring interface %s - loopback", name)
			continue
		}

		// Not a lot else to go on...
		if !strings.HasPrefix(name, "eth") {
			glog.V(2).Infof("Ignoring interface %s - name does not look like ethernet device", name)
			continue
		}

		addrs, err := networkInterface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("error querying network interface %s for IP adddresses: %v", name, err)
		}

		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil {
				return nil, fmt.Errorf("error parsing address %s on network interface %s: %v", addr.String(), name, err)
			}

			if ip.IsLoopback() {
				glog.V(2).Infof("Ignoring address %s (loopback)", ip)
				continue
			}

			if ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() {
				glog.V(2).Infof("Ignoring address %s (link-local)", ip)
				continue
			}

			ips = append(ips, ip)
		}
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("unable to determine pod ip (no adddresses found)")
	}

	if len(ips) != 1 {
		glog.Warningf("Found multiple pod IPs; making arbitrary choice")
		for _, ip := range ips {
			glog.Warningf("\tip: %s", ip.String())
		}
	}
	return ips[0], nil
}
