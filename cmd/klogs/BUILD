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
    name = "klogs",
    srcs = [
        "main.go",
        "root.go",
        "search.go",
        "streams.go",
    ],
    tags = ["automanaged"],
    deps = [
        "@com_github_golang_glog//:go_default_library",
        "@com_github_spf13_cobra//:go_default_library",
        "//pkg/client:go_default_library",
        "//pkg/proto:go_default_library",
    ],
)
