package s3archive

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/golang/glog"
	"kope.io/klog/pkg/archive"
	"kope.io/klog/pkg/proto"
	"net/url"
	"os"
	"path"
	"strings"
)

type Sink struct {
	bucket  string
	basekey string

	s3Client s3iface.S3API
}

var _ archive.Sink = &Sink{}

func NewSink(u *url.URL) (*Sink, error) {
	bucket := strings.TrimSuffix(u.Host, "/")

	s := &Sink{
		bucket:  bucket,
		basekey: u.Path,
	}

	var region string
	{
		config := aws.NewConfig().WithRegion("us-east-1")
		session := session.New()
		s3Client := s3.New(session, config)

		request := &s3.GetBucketLocationInput{}
		request.Bucket = aws.String(bucket)

		glog.V(2).Infof("Querying S3 for bucket location for %q", bucket)
		response, err := s3Client.GetBucketLocation(request)
		if err != nil {
			return nil, fmt.Errorf("error getting location for S3 bucket %q: %v", bucket, err)
		}

		if response.LocationConstraint == nil {
			// US Classic does not return a region
			region = "us-east-1"
		} else {
			region = *response.LocationConstraint
			// Another special case: "EU" can mean eu-west-1
			if region == "EU" {
				region = "eu-west-1"
			}
		}
		glog.V(2).Infof("Found bucket %q in region %q", bucket, region)
	}

	config := aws.NewConfig().WithRegion(region)
	session := session.New()
	s.s3Client = s3.New(session, config)
	return s, nil
}

func (s *Sink) AddToArchive(sourcePath string, podUID string, fileInfo *proto.LogFile) error {
	glog.V(2).Infof("found file to archive: %q %q", sourcePath, fileInfo)
	s3Key := path.Join(s.basekey, "pods", podUID, "logs", fileInfo.Path)

	f, err := os.OpenFile(sourcePath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("unable to open file %q: %v", sourcePath, err)
	}
	defer f.Close()

	request := &s3.PutObjectInput{}
	request.Body = f
	request.Bucket = aws.String(s.bucket)
	request.Key = aws.String(s3Key)

	// We don't need Content-MD5: https://github.com/aws/aws-sdk-go/issues/208

	// TODO: Only if changed?

	_, err = s.s3Client.PutObject(request)
	if err != nil {
		return fmt.Errorf("error writing s3://%s/%s: %v", s.bucket, s3Key, err)
	}
	glog.V(2).Infof("Uploaded file to s3://%s/%s", s.bucket, s3Key)

	return nil
}
