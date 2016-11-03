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
        "factory.go",
        "search.go",
        "streams.go",
    ],
    tags = ["automanaged"],
    deps = [
        "@com_github_golang_glog//:go_default_library",
        "@org_golang_google_grpc//:go_default_library",
        "@org_golang_x_net//context:go_default_library",
        "//pkg/archive:go_default_library",
        "//pkg/archive/s3archive:go_default_library",
        "//pkg/grpc:go_default_library",
        "//pkg/mesh:go_default_library",
        "//pkg/proto:go_default_library",
    ],
)
