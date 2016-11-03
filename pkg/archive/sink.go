package archive

import (
	"kope.io/klog/pkg/proto"
)

type Sink interface {
	AddToArchive(sourcePath string, podUID string, fileInfo *proto.LogFile) error
}
