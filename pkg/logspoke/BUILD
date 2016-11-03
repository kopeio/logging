package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_binary",
    "go_library",
    "go_test",
    "cgo_library",
)

go_library(
    name = "go_default_library",
    srcs = [
        "container_logs.go",
        "localstate.go",
        "log_server.go",
        "log_volumes.go",
        "mesh_member.go",
        "options.go",
        "scraper.go",
    ],
    tags = ["automanaged"],
    deps = [
        "@com_github_golang_glog//:go_default_library",
        "@io_k8s_client_go//pkg/api/v1:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
        "@org_golang_x_net//context:go_default_library",
        "//pkg/archive:go_default_library",
        "//pkg/archive/s3archive:go_default_library",
        "//pkg/proto:go_default_library",
    ],
)
