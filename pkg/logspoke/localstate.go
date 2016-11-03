package logspoke

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io"
	"k8s.io/client-go/pkg/api/v1"
	"kope.io/klog/pkg/archive"
	"kope.io/klog/pkg/proto"
	"math"
	"os"
	"strings"
	"sync"
	"time"
)

const LineBufferSize = 1024 * 1024

const chunkFlushSize = 64 * 1024

var idlePeriod = time.Minute * 15

type NodeState struct {
	nodeFields  *proto.Fields
	archiveSink archive.Sink

	mutex      sync.Mutex
	pods       map[string]*PodState
	containers map[string]*ContainerState
}

type PodState struct {
	nodeState *NodeState
	uid       string

	mutex      sync.Mutex
	streamInfo proto.StreamInfo
	podObject  *v1.Pod
	logs       *LogsState
}

type ContainerState struct {
	nodeState *NodeState
	id        string

	mutex      sync.Mutex
	streamInfo proto.StreamInfo
	logs       *LogsState
	labels     map[string]string
}

type LogsState struct {
	mutex    sync.Mutex
	logs     map[string]*LogFile
	archived map[string]*LogFile
}

type LogFile struct {
	model proto.LogFile
}

func (l *LogFile) canMatch(request *proto.SearchRequest) (bool, []*proto.FieldFilter) {
	if len(request.FieldFilters) == 0 {
		return true, nil
	}

	unmatched := make([]*proto.FieldFilter, 0, len(request.FieldFilters))

	for _, filter := range request.FieldFilters {
		mismatch := false
		processed := false

		// TODO: Well known fields
		if filter.Key == "@timestamp" {
			// TODO: Non-string values
			t, err := time.Parse(time.RFC3339Nano, filter.Value)
			if err != nil {
				glog.Warningf("ignoring error parsing @timestamp value %q", filter.Value)
				continue
			}

			switch filter.Op {
			case proto.FieldFilterOperator_GTE:
				if l.model.MaxTimestamp != 0 && l.model.MaxTimestamp < uint64(t.UnixNano()) {
					mismatch = true
				}

			default:
				glog.Warningf("Unhandled operator: %v", filter)
			}
		} else {
			for _, actual := range l.model.Fields.Fields {
				if filter.Key == actual.Key {
					processed = true

					switch filter.Op {
					case proto.FieldFilterOperator_NOT_EQ:
						if actual.Value == filter.Value {
							mismatch = true
						}
					case proto.FieldFilterOperator_EQ:
						if actual.Value != filter.Value {
							mismatch = true
						}

					default:
						glog.Warningf("Unhandled operator: %v", filter)
					}

					break
				}
			}
		}
		if mismatch {
			return false, nil
		}
		if !processed {
			unmatched = append(unmatched, filter)
		}
	}

	return true, unmatched
}

func newNodeState(archiveSink archive.Sink) *NodeState {
	s := &NodeState{
		archiveSink: archiveSink,
		pods:        make(map[string]*PodState),
		containers:  make(map[string]*ContainerState),
	}
	return s
}

func (s *NodeState) CleanupPodLogs(ids []string) {
	idMap := make(map[string]struct{}, len(ids))
	for _, k := range ids {
		idMap[k] = struct{}{}
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, p := range s.pods {
		func() {
			p.mutex.Lock()
			defer p.mutex.Unlock()

			_, found := idMap[p.uid]
			if !found {
				glog.V(2).Infof("Removing pod logs state: %q", p.uid)
				p.logs = nil
				glog.Warningf("TODO: Remove pods when no state left")
			}
		}()
	}
}

func (s *NodeState) CleanupContainerLogs(ids []string) {
	idMap := make(map[string]struct{}, len(ids))
	for _, k := range ids {
		idMap[k] = struct{}{}
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, p := range s.containers {
		func() {
			p.mutex.Lock()
			defer p.mutex.Unlock()

			_, found := idMap[p.id]
			if !found {
				glog.V(2).Infof("Removing container logs state: %q", p.id)
				p.logs = nil
				glog.Warningf("TODO: Remove containers when no state left")
			}
		}()
	}
}

func (s *NodeState) GetPodState(uid string) *PodState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	pod := s.pods[uid]
	if pod == nil {
		pod = &PodState{
			nodeState: s,
			uid:       uid,
		}
		pod.streamInfo.PodUid = uid
		s.pods[uid] = pod
	}
	return pod
}

func (s *NodeState) GetContainerState(containerid string) *ContainerState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	container := s.containers[containerid]
	if container == nil {
		container = &ContainerState{
			nodeState: s,
			id:        containerid,
		}
		s.containers[containerid] = container
	}
	return container
}

func newLogsState() *LogsState {
	l := &LogsState{
		logs:     make(map[string]*LogFile),
		archived: make(map[string]*LogFile),
	}
	return l
}

func (l *LogsState) foundFile(sourcePath string, relativePath string, stat os.FileInfo, fields *proto.Fields) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	modTime := stat.ModTime()

	logFile := l.logs[sourcePath]
	modified := true
	if logFile != nil {
		if logFile.model.LastModified == modTime.Unix() && logFile.model.Size == stat.Size() {
			modified = false
			glog.V(4).Infof("File not modified: %q", sourcePath)
		}
	} else {
		logFile = &LogFile{
			model: proto.LogFile{
				Path:         relativePath,
				LastModified: modTime.Unix(),
				Size:         stat.Size(),
				Fields:       fields,
			},
		}
		l.logs[sourcePath] = logFile
	}

	if modified {
		_, maxTimestamp, err := findMaxTimestamp(sourcePath)
		if err != nil {
			glog.Warningf("error finding max timestamp for %q: %v", sourcePath, err)
			logFile.model.MaxTimestamp = 0
		} else {
			logFile.model.MaxTimestamp = uint64(maxTimestamp)
		}
	}

	return nil
}

func (p *ContainerState) foundFile(sourcePath string, relativePath string, stat os.FileInfo, fields *proto.Fields) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.logs == nil {
		p.logs = newLogsState()
	}

	return p.logs.foundFile(sourcePath, relativePath, stat, fields)
}

func (p *PodState) foundFile(sourcePath string, relativePath string, stat os.FileInfo, fields *proto.Fields) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.logs == nil {
		p.logs = newLogsState()
	}

	// TODO: Move to shared LogState.foundFile code; move archiving elsewhere
	modTime := stat.ModTime()
	now := time.Now()

	if modTime.Add(idlePeriod).Before(now) {
		logFile := p.logs.logs[sourcePath]
		modified := false
		if logFile != nil {
			if logFile.model.LastModified == modTime.Unix() && logFile.model.Size == stat.Size() {
				glog.V(4).Infof("File not modified: %q", sourcePath)
			} else {
				modified = true
			}
		} else {
			logFile = &LogFile{
				model: proto.LogFile{
					Path:         relativePath,
					LastModified: modTime.Unix(),
					Size:         stat.Size(),
					Fields:       fields,
				},
			}
			p.logs.logs[sourcePath] = logFile
			modified = true
		}

		if modified && p.nodeState.archiveSink != nil {
			archived := p.logs.archived[relativePath]
			if archived == nil || *archived != *logFile {
				glog.Warningf("Should not hold lock while archiving file")
				err := p.nodeState.archiveSink.AddToArchive(sourcePath, p.uid, &logFile.model)
				if err != nil {
					glog.Warningf("error adding file %q to archive: %v", sourcePath, err)
				} else {
					p.logs.archived[relativePath] = logFile
				}
			}
		}
	}
	return nil
}

var _ proto.LogServerServer = &NodeState{}

func (s *NodeState) GetStreams(request *proto.GetStreamsRequest, out proto.LogServer_GetStreamsServer) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	glog.V(2).Infof("GetStreamsRequest %q", request)
	for _, p := range s.pods {
		err := func() error {
			p.mutex.Lock()
			defer p.mutex.Unlock()

			err := out.Send(&p.streamInfo)
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	for _, p := range s.containers {
		err := func() error {
			p.mutex.Lock()
			defer p.mutex.Unlock()

			err := out.Send(&p.streamInfo)
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}

	return nil
}

type fileScanOperation struct {
	sourcePath string
	fields     *proto.Fields
	unmatched  []*proto.FieldFilter
}

func (s *NodeState) Search(request *proto.SearchRequest, out proto.LogServer_SearchServer) error {
	glog.Warningf("TODO: Scan files before search?")

	var ops []*fileScanOperation

	func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		glog.V(2).Infof("Search %q", request)
		for _, p := range s.pods {
			func() {
				p.mutex.Lock()
				defer p.mutex.Unlock()

				if p.logs != nil {
					for k, l := range p.logs.logs {
						canMatch, unmatched := l.canMatch(request)
						if !canMatch {
							continue
						}
						ops = append(ops, &fileScanOperation{
							sourcePath: k,
							fields:     l.model.Fields,
							unmatched:  unmatched,
						})
					}
				}
			}()
		}

		for _, p := range s.containers {
			func() {
				p.mutex.Lock()
				defer p.mutex.Unlock()

				if p.logs != nil {
					for k, l := range p.logs.logs {
						canMatch, unmatched := l.canMatch(request)
						if !canMatch {
							glog.V(2).Infof("Excluded file %s size=%d maxTimestamp=%d %v", k, l.model.Size, l.model.MaxTimestamp, l.model.Fields)
							continue
						}
						// TODO: Skip if size 0? ... maybe only if file is "closed"

						glog.V(2).Infof("Unable to exclude file %s size=%d maxTimestamp=%d %v", k, l.model.Size, l.model.MaxTimestamp, l.model.Fields)
						ops = append(ops, &fileScanOperation{
							sourcePath: k,
							fields:     l.model.Fields,
							unmatched:  unmatched,
						})
					}
				}
			}()
		}
	}()

	// TODO: Build callback class
	var matchBytes []byte
	if request.Contains != "" {
		glog.Warningf("JSON match encoding not yet implemented")
		matchBytes = []byte(request.Contains)
	}
	matcher := func(line []byte) bool {
		if matchBytes != nil {
			if bytes.Index(line, matchBytes) == -1 {
				return false
			}
		}
		return true
	}

	buffer := make([]byte, LineBufferSize, LineBufferSize)
	for _, l := range ops {
		err := l.searchLogFile(buffer, matcher, request, out)
		if err != nil {
			glog.Warningf("error searching log file %q: %v", l, err)
			return fmt.Errorf("error searching log file %q: %v", l, err)
		}
	}
	return nil
}

type dockerLine struct {
	Log    string `json:"log,omitempty"`
	Stream string `json:"stream,omitempty"`
	Time   string `json:"time,omitempty"`
}

func (s *fileScanOperation) searchLogFile(buffer []byte, matcher func([]byte) bool, request *proto.SearchRequest, out proto.LogServer_SearchServer) error {
	glog.V(2).Infof("search log file %q: %v", s.sourcePath, s.unmatched)

	// TODO: Skip if size 0?

	var in io.Reader
	f, err := os.OpenFile(s.sourcePath, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			glog.V(2).Infof("ignoring log file that no longer exists %q", s.sourcePath)
			return nil
		} else {
			glog.Warningf("ignoring error opening log file %q: %v", s.sourcePath, err)
			return nil
		}
	}
	defer f.Close()

	in = f
	if strings.HasSuffix(s.sourcePath, ".gz") {
		gz, err := gzip.NewReader(in)
		if err != nil {
			return fmt.Errorf("error building gzip decompressor for %q: %v", s.sourcePath, err)
		}
		defer gz.Close()
		in = gz
	}

	var chunk *proto.SearchResultChunk
	var chunkSize int

	scanner := bufio.NewScanner(in)
	scanner.Buffer(buffer, cap(buffer))
	for scanner.Scan() {
		line := scanner.Bytes()

		if !matcher(line) {
			continue
		}
		if chunk == nil {
			chunk = &proto.SearchResultChunk{}
			chunkSize = 32
			chunk.CommonFields = s.fields
		}

		item := &proto.SearchResult{}
		item.Raw = line

		var l dockerLine
		err = json.Unmarshal(line, &l)
		if err == nil {
			fields := item.Fields
			if fields == nil {
				fields = &proto.Fields{}
				item.Fields = fields
			}
			if l.Log != "" {
				fields.Fields = append(fields.Fields, &proto.Field{
					Key:   "log",
					Value: l.Log,
				})
				chunkSize += 8 + len(l.Log)
			}
			if l.Stream != "" {
				fields.Fields = append(fields.Fields, &proto.Field{
					Key:   "stream",
					Value: l.Stream,
				})
				chunkSize += 8 + len(l.Stream)
			}
			if l.Time != "" {
				t, err := time.Parse(time.RFC3339Nano, l.Time)
				if err == nil {
					item.Timestamp = uint64(t.UnixNano())
				}
				chunkSize += 10
			}
		}

		if len(s.unmatched) != 0 {
			itemFields := item.Fields
			if itemFields == nil {
				continue
			}

			match := true
			for _, filter := range s.unmatched {
				// TODO: Well known fields
				if filter.Key == "@timestamp" {
					// TODO: What if no timestamp?

					// TODO: Non-string values
					t, err := time.Parse(time.RFC3339Nano, filter.Value)
					if err != nil {
						glog.Warningf("ignoring error parsing @timestamp value %q", filter.Value)
						continue
					}

					switch filter.Op {
					case proto.FieldFilterOperator_GTE:
						if !(item.Timestamp >= uint64(t.UnixNano())) {
							match = false
						}

					default:
						glog.Warningf("Unhandled operator: %v", filter)
					}
				} else {
					found := false
					for _, actual := range itemFields.Fields {
						if actual.Key == filter.Key {
							found = true
							switch filter.Op {
							case proto.FieldFilterOperator_NOT_EQ:
								if actual.Value == filter.Value {
									match = false
								}
							case proto.FieldFilterOperator_EQ:
								if actual.Value != filter.Value {
									match = false
								}

							default:
								glog.Warningf("Unhandled operator: %v", filter)
							}
							break
						}
					}
					if !found {
						match = false
					}
				}

				if !match {
					break
				}
			}

			if !match {
				continue
			}
		}

		chunk.Items = append(chunk.Items, item)
		chunkSize += 8 + len(line)

		if chunkSize > chunkFlushSize {
			if err := out.Send(chunk); err != nil {
				return err
			}
			chunk = nil
		}
	}

	if chunk != nil {
		if err := out.Send(chunk); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		glog.Warningf("error reading log file %q: %v", s.sourcePath, err)
	}
	return nil
}

func findMaxTimestamp(sourcePath string) (uint64, uint64, error) {
	buffer := make([]byte, LineBufferSize, LineBufferSize)

	glog.V(2).Infof("findMaxTimestamp for %q", sourcePath)

	var in io.Reader
	f, err := os.OpenFile(sourcePath, os.O_RDONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			glog.V(2).Infof("ignoring log file that no longer exists %q", sourcePath)
			return 0, 0, err
		} else {
			return 0, 0, err
		}
	}
	defer f.Close()

	// TODO: rotate, attach metadata
	glog.Warningf("findMaxTimestamp is very inefficient")

	in = f
	if strings.HasSuffix(sourcePath, ".gz") {
		gz, err := gzip.NewReader(in)
		if err != nil {
			return 0, 0, fmt.Errorf("error building gzip decompressor for %q: %v", sourcePath, err)
		}
		defer gz.Close()
		in = gz
	}

	minTimestamp := uint64(math.MaxUint64)
	maxTimestamp := uint64(0)
	scanner := bufio.NewScanner(in)
	scanner.Buffer(buffer, cap(buffer))
	for scanner.Scan() {
		line := scanner.Bytes()

		// TODO: Don't bother parsing unless we are at the end?
		var l dockerLine
		err = json.Unmarshal(line, &l)
		if err == nil {
			if l.Time != "" {
				t, err := time.Parse(time.RFC3339Nano, l.Time)
				if err == nil {
					ts := uint64(t.UnixNano())
					if ts > maxTimestamp {
						maxTimestamp = ts
					}
					if ts < minTimestamp {
						minTimestamp = ts
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, 0, fmt.Errorf("error reading log file %q: %v", sourcePath, err)
	}

	return minTimestamp, maxTimestamp, nil
}
