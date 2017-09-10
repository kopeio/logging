package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_binary",
    "go_library",
)

go_binary(
    name = "klogs",
    library = ":go_default_library",
    tags = ["automanaged"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "main.go",
        "passwordflag.go",
        "root.go",
        "search.go",
        "streams.go",
    ],
    visibility = ["//visibility:private"],
    deps = [
        "//pkg/client:go_default_library",
        "@com_github_golang_glog//:go_default_library",
        "@com_github_spf13_cobra//:go_default_library",
    ],
)
