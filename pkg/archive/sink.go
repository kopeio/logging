package archive

import (
	"kope.io/klogs/pkg/proto"
)

type Sink interface {
	AddToArchive(sourcePath string, podUID string, fileInfo *proto.LogFile) error
}
