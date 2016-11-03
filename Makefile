# TODO: Move entirely to bazel?
.PHONY: images

protobuf:
	protoc -I ./pkg/proto ./pkg/proto/log.proto --go_out=plugins=grpc:pkg/proto

gofmt:
	gofmt -w -s cmd/
	gofmt -w -s pkg/

push: images
	docker push kopeio/klog-spoke:latest
	docker push kopeio/klog-hub:latest

images:
	bazel run //images:klog-spoke kopeio/klog-spoke:latest
	bazel run //images:klog-hub kopeio/klog-hub:latest
