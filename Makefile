# TODO: Move entirely to bazel?
.PHONY: images

protobuf:
	protoc -I ./pkg/proto ./pkg/proto/log.proto --go_out=plugins=grpc:pkg/proto

gofmt:
	gofmt -w -s cmd/
	gofmt -w -s pkg/

push: images
	docker push kopeio/logging-spoke:latest
	docker push kopeio/logging-hub:latest

images:
	bazel run //images:klog-spoke
	docker tag bazel/images:klog-spoke kopeio/logging-spoke:latest
	bazel run //images:klog-hub
	docker tag bazel/images:klog-hub kopeio/logging-hub:latest
