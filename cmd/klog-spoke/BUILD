package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_binary",
    "go_library",
    "go_test",
    "cgo_library",
)

go_binary(
    name = "klog-spoke",
    srcs = ["main.go"],
    tags = ["automanaged"],
    deps = [
        "@com_github_golang_glog//:go_default_library",
        "@com_github_spf13_pflag//:go_default_library",
        "//pkg/logspoke:go_default_library",
    ],
)
