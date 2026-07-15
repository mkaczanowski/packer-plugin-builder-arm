# Packer Plugin Builder ARM

[![Build Status][github-badge]][github]
[![GoDoc][godoc-badge]][godoc]
[![GoReportCard][report-badge]][report]
[![Docker Pulls][docker-pulls]][docker-hub]
[![Docker Image Size][docker-size]][docker-hub]
[![Docker Image Version][docker-version]][docker-hub]

[github-badge]:https://img.shields.io/github/actions/workflow/status/mkaczanowski/packer-plugin-builder-arm/docker.yml?branch=master
[github]: https://github.com/mkaczanowski/packer-plugin-builder-arm/actions
[godoc-badge]: https://godoc.org/github.com/mkaczanowski/packer-plugin-builder-arm?status.svg
[godoc]: https://godoc.org/github.com/mkaczanowski/packer-plugin-builder-arm
[report-badge]: https://goreportcard.com/badge/github.com/mkaczanowski/packer-plugin-builder-arm
[report]: https://goreportcard.com/report/github.com/mkaczanowski/packer-plugin-builder-arm
[docker-hub]: https://hub.docker.com/r/mkaczanowski/packer-plugin-builder-arm
[docker-pulls]: https://img.shields.io/docker/pulls/mkaczanowski/packer-plugin-builder-arm
[docker-size]: https://img.shields.io/docker/image-size/mkaczanowski/packer-plugin-builder-arm
[docker-version]: https://img.shields.io/docker/v/mkaczanowski/packer-plugin-builder-arm?sort=semver


This plugin allows you to build or extend ARM system image. It operates in two modes:
* new - creates empty disk image and populates the rootfs on it
* reuse - uses already existing image as the base
* resize - uses already existing image but resize given partition (ie. root)

Plugin mimics standard image creation process, such as:
* building base empty image (dd)
* partitioning (sgdisk / sfdisk)
* filesystem creation (mkfs.type)
* partition mapping (losetup)
* filesystem mount (mount)
* populate rootfs (tar/unzip/xz etc)
* setup qemu + chroot
* customize installation within chroot

The virtualization works via [binfmt_misc](https://en.wikipedia.org/wiki/Binfmt_misc) kernel feature and qemu.

Since the setup varies a lot for different hardware types, the example configuration is available per "board". Currently the following boards are supported (feel free to add more):
* bananapi-r1 (Archlinux ARM)
* beaglebone-black (Archlinux ARM, Debian)
* jetson-nano (Ubuntu)
* odroid-u3 (Archlinux ARM)
* odroid-xu4 (Archlinux ARM, Ubuntu)
* parallella (Ubuntu)
* raspberry-pi (Archlinux ARM, Raspbian)
* raspberry-pi-3 (Archlinux ARM (armv8))
* raspberry-pi-4 (Archlinux ARM (armv7), Ubuntu 20.04 LTS))
* wandboard (Archlinux ARM)
* armv7 generic (Alpine, Archlinux ARM)

# Quick start
```
git clone https://github.com/mkaczanowski/packer-plugin-builder-arm
cd packer-plugin-builder-arm
go mod download
go build -o packer-plugin-builder-arm

# register the locally-built plugin with packer (one-time, or whenever you rebuild)
packer plugins install --path ./packer-plugin-builder-arm github.com/mkaczanowski/arm

sudo -E packer build boards/odroid-u3/archlinuxarm.json
```

Since this is a multi-component plugin, HCL2 templates should declare it in a `required_plugins` block so `packer init` knows what version is expected:
```hcl
packer {
  required_plugins {
    arm = {
      source  = "github.com/mkaczanowski/arm"
      version = ">= 1.0.9"
    }
  }
}
```
Legacy JSON templates (`"type": "arm"` builders) don't support `required_plugins`, but work the same way once the plugin has been installed with `packer plugins install`.
## Run in Docker
This method is primarily for macOS users where is no native way to use qemu-user-static, loop mount Linux specific filesystems and install all above mentioned Linux specific tools (or Linux users, who do not want to setup packer and all the tools).

The container is a multi-arch container (linux/amd64 or linux/arm64), that can be used on Intel (x86_64) or Apple M1 (arm64) Macs and also on Linux machines running linux (x86_64 or aarch64) kernels.

> **_NOTE:_** On Macs: Don't run `go build .` (that produces a **darwin** binary) and then run below `docker run ...` commands from the same folder to avoid the error `error initializing builder 'arm': fork/exec /build/packer-plugin-builder-arm: exec
format error` (**linux** packer process within docker fails to load the outside container compiled packer-plugin-builder-arm binary due to being a **darwin** binary). Delete any local binary via `rm -r packer-*` to solely use the binary already included and provided by the container.

### Usage via container from Docker Hub:

Pull the latest version of the container to ensure the next commands are not using an old cached version of the container :
```
docker pull mkaczanowski/packer-plugin-builder-arm:latest
```

Build a board:
```
docker run --rm --privileged -v /dev:/dev -v ${PWD}:/build mkaczanowski/packer-plugin-builder-arm:latest build boards/raspberry-pi/raspbian.json
```
Build a board with more system packages (e.g. bmap-tools, zstd) can be added via the parameter `-extra-system-packages=...`:
```
docker run --rm --privileged -v /dev:/dev -v ${PWD}:/build mkaczanowski/packer-plugin-builder-arm:latest build boards/raspberry-pi/raspbian.json -extra-system-packages=bmap-tools,zstd
```

> **_NOTE:_** In above commands **latest** can also be replaced via e.g. **1.0.3** to get a specific container version.

### Usage via local container build (supports amd64/aarch64 hosts):
Build the container locally:
```
docker build -t packer-plugin-builder-arm -f docker/Dockerfile .
```
Run packer via the local built container:
```
docker run --rm --privileged -v /dev:/dev -v ${PWD}:/build packer-plugin-builder-arm build boards/raspberry-pi/raspbian.json
```

# Dependencies
* `sfdisk / sgdisk`
* `e2fsprogs`
* `parted` (resize mode)
* `resize2fs` (resize mode)
* `qemu-img` (resize mode)

# Configuration
Configuration is split into 3 parts:
* remote file config
* image config
* qemu config

## Remote file
Describes the remote file that is going to be used as base image or rootfs archive (depending on `image_build_method`)

```
"file_urls" : ["http://os.archlinuxarm.org/os/ArchLinuxARM-odroid-xu3-latest.tar.gz"],
"file_checksum_url": "http://hu.mirror.archlinuxarm.org/os/ArchLinuxARM-odroid-xu3-latest.tar.gz.md5",
"file_checksum_type": "md5",
"file_unarchive_cmd": ["bsdtar", "-xpf", "$ARCHIVE_PATH", "-C", "$MOUNTPOINT"],
"file_target_extension": "tar.gz",
```

Downloads of the `file_urls` are done with the help of `github.com/hashicorp/go-getter`, which supports various protocols: local files, http(s) and various others, see https://github.com/hashicorp/go-getter#supported-protocols-and-detectors). Downloading via more protocols can be done by using other tools (curl, wget, rclone, ...) before running packer and referencing the downloaded files as local file in `file_urls`.

The `file_unarchive_cmd` is optional and should be used if the standard golang archiver can't handle the archive format.

Raw images format (`.img` or `.iso`) can be used by defining the `file_target_extension` appropriately.

Some mirrors reject requests carrying the default HTTP client User-Agent. In that case, set `file_user_agent` to override it, e.g. `"file_user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0"`.

## Image config
The base image description (size, partitions, mountpoints etc).

```
"image_build_method": "new",
"image_path": "odroid-xu4.img",
"image_size": "2G",
"image_type": "dos",
"image_partitions": [
    {
        "name": "root",
        "type": "8300",
        "start_sector": "4096",
        "filesystem": "ext4",
        "size": "0",
        "mountpoint": "/"
    }
],
```

The plugin doesn't try to detect the image partitions because that varies a lot. Instead it solely depend on `image_partitions` specification, so you should set that even if you reuse the image (`method` = reuse).

## Qemu config
Anything qemu related:

```
"qemu_binary_source_path": "/usr/bin/qemu-arm-static",
"qemu_binary_destination_path": "/usr/bin/qemu-arm-static"
```

The arm instruction set (default=`armv7l` for qemu-arm-static) to be emulated can be defined via the `QEMU_CPU` variable. To switch to `armv6l` (check with `uname -m` as provission command) run packer e.g. via:
* `QEMU_CPU=arm1176 packer build ...`
* `docker run -e QEMU_CPU=arm1176 ...`

# Chroot provisioner
To execute command within chroot environment you should use chroot communicator:
```
"provisioners": [
 {
   "type": "shell",
   "inline": [
     "pacman-key --init",
     "pacman-key --populate archlinuxarm"
   ]
 }
]
```
This plugin doesn't resize partitions on the base image. However, you can easily expand partition size at the boot time with a systemd service. [Here](./boards/raspberry-pi/archlinuxarm.json) you can find real-life example, where a raspberry pi root-fs partition expands to all available space on sdcard.

# Flashing
To dump image on device you can use [custom postprocessor](https://github.com/mkaczanowski/packer-post-processor-flasher) (really wrapper around `dd` with some sanity checks):
```
"post-processors": [
 {
     "type": "flasher",
     "device": "/dev/sdX",
     "block_size": "4096",
     "interactive": true
 }
]
```

# Other
## Generating rootfs archive
While image (`.img`) format is useful for most cases, you might want to use
rootfs for other purposes (ex. export to docker). This is how you can generate
rootfs archive instead of image:
```
"image_path": "odroid-xu4.img" # generates image
"image_path": "odroid-xu4.img.tar.gz" # generates rootfs archive
```

## Resizing image
Currently resizing is only limited to expanding single `ext{2,3,4}` partition with `resize2fs`. This is often requested feature where already built image is given and we need to expand the main partition to accommodate changes made in provisioner step (ie. installing packages).

To resize a partition you need to set `image_build_method` to `resize` mode and set selected partition size to `0`, for example:
```
"builders": [
  {
    "type": "arm",
    "image_build_method": "resize",
    "image_partitions": [
      {
        "name": "boot",
        ...
      },
      {
        "name": "root",
        "size": "0",
        ...
      }
    ],
    ...
  }
]
```

Complete examples:

- [`boards/raspberry-pi/raspbian-resize.json`](./boards/raspberry-pi/raspbian-resize.json)
- [`boards/beaglebone-black/ubuntu.pkr.hcl`](./boards/beaglebone-black/ubuntu.pkr.hcl)

## Docker
With `artifice` plugin you can pass rootfs archive to docker plugins
```
"post-processors": [
    [{
        "type": "artifice",
        "files": ["rootfs.tar.gz"]
    },
    {
        "type": "docker-import",
        "repository": "mkaczanowski/archlinuxarm",
        "tag": "latest"
    }],
    ...
]
```

## CI/CD
This is the live example on how to use github actions to push image to docker image registry:
```
cat .github/workflows/archlinuxarm-armv7-docker.yml
```

## How is this plugin different from `solo-io/packer-builder-arm-image`
https://github.com/hashicorp/packer/pull/8462

# Examples
For more examples please see:
```
tree boards/
```

The repository also includes some arm typical scripts to e.g. resize partitions on first boot or more extensive
provision scripts:

```
tree scripts/
```

A big resource for packer provisions scripts is the [GitHub Actions runner images](https://github.com/actions/runner-images) repository.

# Troubleshooting
Many of the reported issues are platform/OS specific. If you happen to have
problems, the first question you should ask yourself is:
> Is my setup faulty? or is there an actual issue?

To answer that question, I'd recommend reproducing the error on the VM, for
instance:
```
cd packer-plugin-builder-arm
vagrant up
vagrant provision
```
> Note: For this the disksize plugin is needed if not already installed `vagrant plugin install vagrant-disksize`

# Demo
[![asciicast](https://asciinema.org/a/7ad1nm2Q7DRFVlHpqAknPolNo.svg)](https://asciinema.org/a/7ad1nm2Q7DRFVlHpqAknPolNo)
