package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_binary",
    "go_library",
)

go_binary(
    name = "klog-hub",
    library = ":go_default_library",
    tags = ["automanaged"],
)

go_library(
    name = "go_default_library",
    srcs = ["main.go"],
    visibility = ["//visibility:private"],
    deps = [
        "//pkg/grpc:go_default_library",
        "//pkg/loghub:go_default_library",
        "@com_github_golang_glog//:go_default_library",
        "@com_github_spf13_pflag//:go_default_library",
    ],
)
