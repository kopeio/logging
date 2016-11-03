package(default_visibility = ["//visibility:public"])

load("@bazel_tools//tools/build_defs/docker:docker.bzl", "docker_build")

docker_build(
    name = "busybox",
    debs = [
        "@busybox_deb//file",
    ],
    symlinks = {
        "/bin/sh": "/bin/busybox",
        "/usr/bin/busybox": "/bin/busybox",
        "/usr/sbin/busybox": "/bin/busybox",
        "/sbin/busybox": "/bin/busybox",
    },
)

docker_build(
    name = "busybox-libc",
    base = ":busybox",
    debs = [
        "@libc_deb//file",
    ],
)

docker_build(
    name = "klog-spoke",
    base = ":busybox-libc",
    cmd = ["/klog-spoke"],
    files = [
        "//cmd/klog-spoke",
    ],
    repository = "kopeio",
)

docker_build(
    name = "klog-hub",
    base = ":busybox-libc",
    cmd = ["/klog-hub"],
    files = [
        "//cmd/klog-hub",
    ],
    repository = "kopeio",
)
