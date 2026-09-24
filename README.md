# Packer Plugin Builder ARM

[![Build Status][github-badge]][github]
[![GoDoc][godoc-badge]][godoc]
[![Docker Pulls][docker-pulls]][docker-hub]
[![Docker Image Size][docker-size]][docker-hub]
[![Docker Image Version][docker-version]][docker-hub]

[github-badge]:https://img.shields.io/github/actions/workflow/status/mkaczanowski/packer-plugin-builder-arm/docker.yml?branch=master
[github]: https://github.com/mkaczanowski/packer-plugin-builder-arm/actions
[godoc-badge]: https://godoc.org/github.com/mkaczanowski/packer-plugin-builder-arm?status.svg
[godoc]: https://godoc.org/github.com/mkaczanowski/packer-plugin-builder-arm
[docker-hub]: https://hub.docker.com/r/mkaczanowski/packer-plugin-builder-arm
[docker-pulls]: https://img.shields.io/docker/pulls/mkaczanowski/packer-plugin-builder-arm
[docker-size]: https://img.shields.io/docker/image-size/mkaczanowski/packer-plugin-builder-arm
[docker-version]: https://img.shields.io/docker/v/mkaczanowski/packer-plugin-builder-arm?sort=semver


This plugin allows you to build or extend ARM system images. It operates in three modes:
* new - creates empty disk image and populates the rootfs on it
* reuse - uses already existing image as the base
* resize - uses already existing image but resize given partition (ie. root)

Plugin mimics standard image creation process, such as:
* building base empty image (dd)
* partitioning (sgdisk / sfdisk)
* filesystem creation (mkfs.type)
* partition mapping (losetup / kpartx)
* filesystem mount (mount)
* populate rootfs (tar/unzip/xz etc)
* setup qemu + chroot
* customize installation within chroot

The virtualization works via [binfmt_misc](https://en.wikipedia.org/wiki/Binfmt_misc) kernel feature and qemu.

Since the setup varies a lot for different hardware types, the example configuration is available per "board". Currently the following boards are supported (feel free to add more):
* bananapi-r1 (Archlinux ARM)
* beaglebone-black (Angstrom, Archlinux ARM, Debian)
* jetson-nano (Ubuntu)
* odroid-u3 (Archlinux ARM)
* odroid-xu4 (Archlinux ARM, Ubuntu)
* parallella (Archlinux ARM, Ubuntu)
* raspberry-pi (Archlinux ARM, Raspbian, Raspberry Pi OS Lite)
* raspberry-pi-3 (Archlinux ARM (armv8), Raspberry Pi OS Lite (arm64))
* raspberry-pi-4 (Archlinux ARM (armv8), Ubuntu 20.04 LTS)
* rock-4b (Debian, via Radxa debos image)
* wandboard (Archlinux ARM)
* armv7 generic (Alpine, Archlinux ARM)
* armv8 generic (Archlinux ARM)

# Quick start
```bash
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
This method is primarily for macOS users, where there is no native way to use qemu-user-static, loop mount Linux specific filesystems and install all above mentioned Linux specific tools (or for Linux users, who do not want to set up packer and all the tools).

The container is a multi-arch container (linux/amd64 or linux/arm64), that can be used on Intel (x86_64) or Apple M1 (arm64) Macs and also on Linux machines running linux (x86_64 or aarch64) kernels.

> **_NOTE:_** On Macs: Don't run `go build .` (that produces a **darwin** binary) and then run below `docker run ...` commands from the same folder to avoid the error `error initializing builder 'arm': fork/exec /build/packer-plugin-builder-arm: exec
format error` (**linux** packer process within docker fails to load the outside container compiled packer-plugin-builder-arm binary due to being a **darwin** binary). Delete any local binary via `rm -r packer-*` to solely use the binary already included and provided by the container.

### Usage via container from Docker Hub:

Pull the latest version and capture its immutable digest:
```bash
IMAGE=mkaczanowski/packer-plugin-builder-arm:latest
docker pull "${IMAGE}"
VERIFIED_IMAGE=$(docker image inspect "${IMAGE}" --format '{{index .RepoDigests 0}}')
```

The published images are signed with [cosign](https://docs.sigstore.dev/cosign/signing/overview/) (keyless, via GitHub Actions OIDC). Since the container runs with `--privileged` and `-v /dev:/dev`, verify the immutable image before running it:
```bash
cosign verify "${VERIFIED_IMAGE}" \
  --certificate-identity-regexp='^https://github\.com/mkaczanowski/packer-plugin-builder-arm/\.github/workflows/docker\.yml@refs/(heads/master|tags/v[^/]+)$' \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com
```

Build a board using the verified digest:
```bash
docker run --rm --privileged -v ${PWD}:/build "${VERIFIED_IMAGE}" build boards/raspberry-pi/raspbian.json
```
Build a board with more system packages (e.g. bmap-tools, zstd) can be added via the parameter `-extra-system-packages=...`:
```bash
docker run --rm --privileged -v ${PWD}:/build "${VERIFIED_IMAGE}" build boards/raspberry-pi/raspbian.json -extra-system-packages=bmap-tools,zstd
```

> **_NOTE:_** In the `IMAGE` variable, **latest** can also be replaced via e.g. **1.0.3** to get a specific container version.

The images also carry build provenance attestations (source repo, commit, builder), inspectable via:
```bash
docker buildx imagetools inspect mkaczanowski/packer-plugin-builder-arm:latest --format "{{ json .Provenance }}"
```

### Usage via local container build (supports amd64/aarch64 hosts):
Build the container locally:
```bash
docker build -t packer-plugin-builder-arm -f docker/Dockerfile .
```
Run packer via the local built container:
```bash
docker run --rm --privileged -v ${PWD}:/build packer-plugin-builder-arm build boards/raspberry-pi/raspbian.json
```

# Dependencies
* `sfdisk / sgdisk`
* `kpartx`
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

```json
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

```json
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

The plugin doesn't try to detect the image partitions because that varies a lot. Instead it solely depends on the `image_partitions` specification, so you should set that even if you reuse the image (`image_build_method` = `reuse`).

### Filesystem labels and UUIDs
In `new` mode, `filesystem_make_options` passes flags to `mkfs`. For FAT, use
`filesystem = "fat"` with `filesystem_make_options = ["-n", "BOOT"]`. For
ext4, `filesystem_make_options = ["-L", "ROOT", "-U", "<unique-uuid>"]` sets
both values. See the `mkfs.vfat`/`mke2fs` man pages for other flags.

`reuse` and `resize` preserve existing identifiers. Filesystem UUIDs are not
partition-table `PARTUUID`s; change the DOS/MBR disk ID with
`sfdisk --disk-id`, or a GPT partition GUID with
`sfdisk --part-uuid`/`sgdisk --partition-guid`.

## Qemu config
Anything qemu related:

```json
"qemu_binary_source_path": "/usr/bin/qemu-arm-static",
"qemu_binary_destination_path": "/usr/bin/qemu-arm-static"
```

Note: Ubuntu 26.04 (resolute) merged `qemu-user-static` into `qemu-user` and
renamed the binaries to `qemu-<arch>` (e.g. `/usr/bin/qemu-arm`). On such
hosts install `qemu-user` and `qemu-user-binfmt`, then point
`qemu_binary_source_path` at the new name. `qemu_binary_destination_path` can
stay as is; it only names the file inside the image and must match the binfmt
registration, which still uses the `-static` names.

The arm instruction set (default=`armv7l` for qemu-arm-static) to be emulated can be defined via the `QEMU_CPU` variable. To switch to `armv6l` (check with `uname -m` as a provisioner command) run packer e.g. via:
* `QEMU_CPU=arm1176 packer build ...`
* `docker run -e QEMU_CPU=arm1176 ...`

# Chroot provisioner
To execute command within chroot environment you should use chroot communicator:
```json
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

## DNS (resolv.conf) inside the chroot
The chroot shares the host kernel but not its `/etc/resolv.conf`, so DNS is
often broken inside provisioners (see [#144](https://github.com/mkaczanowski/packer-plugin-builder-arm/issues/144)).
On many images the file is also a dangling symlink (e.g. systemd-resolved
pointing into `/run`), which is why bind-mounting the host file via
`image_chroot_mounts` fails with `mkdir .../etc/resolv.conf: file exists` —
that mechanism only works for directories.

Handle it in your provisioners instead: replace the file for the duration of
the build and restore it afterwards, e.g. with a static nameserver (as done in
[boards/raspberry-pi/archlinuxarm.json](./boards/raspberry-pi/archlinuxarm.json)):

```hcl
provisioner "shell" {
  inline = [
    "mv /etc/resolv.conf /etc/resolv.conf.bak || true",
    "echo 'nameserver 8.8.8.8' > /etc/resolv.conf",
  ]
}

# ... your provisioners ...

provisioner "shell" {
  inline = [
    "rm -f /etc/resolv.conf",
    "mv /etc/resolv.conf.bak /etc/resolv.conf || true",
  ]
}
```

Or copy the host resolver configuration in and restore the original file at
the end:

```hcl
provisioner "shell" {
  inline = ["mv /etc/resolv.conf /etc/resolv.conf.bak || true"]
}
provisioner "file" {
  source      = "/etc/resolv.conf"
  destination = "/etc/resolv.conf"
}

# ... your provisioners ...

provisioner "shell" {
  inline = [
    "rm -f /etc/resolv.conf",
    "mv /etc/resolv.conf.bak /etc/resolv.conf || true",
  ]
}
```

Skip the restore step if the image should keep working DNS settings.

## System services inside the chroot
The chroot has no target systemd, D-Bus, or NetworkManager daemon. Commands that
need them, including the Ansible `nmcli` module, fail.

NetworkManager 1.42 and newer can create a keyfile offline:

```bash
umask 077
nmcli --offline connection add type wifi con-name my-wifi ssid my-ssid \
  wifi-sec.key-mgmt wpa-psk wifi-sec.psk my-password \
  > /etc/NetworkManager/system-connections/my-wifi.nmconnection
```

For older versions, write a root-owned `0600` keyfile directly; include
`security=802-11-wireless-security` under `[wifi]` and credentials under
`[wifi-security]`. Runtime commands such as `systemctl start`, `hostnamectl`,
and `timedatectl` still need their daemons. `systemctl enable UNIT` works in a
chroot; it is safer than hand-written links because it applies every
`[Install]` directive. From the host, use
`systemctl --root=/tmp/<mountpoint> enable UNIT`.

This plugin doesn't resize partitions on the base image. However, you can easily expand partition size at the boot time with a systemd service. [Here](./boards/raspberry-pi/archlinuxarm.json) you can find real-life example, where a raspberry pi root-fs partition expands to all available space on sdcard.

# Flashing
To dump image on device you can use [custom postprocessor](https://github.com/mkaczanowski/packer-post-processor-flasher) (really wrapper around `dd` with some sanity checks):
```json
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
```json
"image_path": "odroid-xu4.img" # generates image
"image_path": "odroid-xu4.img.tar.gz" # generates rootfs archive
```

## Resizing image
Currently resizing is only limited to expanding single `ext{2,3,4}` partition with `resize2fs`. This is often requested feature where already built image is given and we need to expand the main partition to accommodate changes made in provisioner step (ie. installing packages).

To resize a partition you need to set `image_build_method` to `resize` mode and set selected partition size to `0`, for example:
```json
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

Complete example:

- [`boards/raspberry-pi/raspbian-resize.json`](./boards/raspberry-pi/raspbian-resize.json)

Notes:
* Resize only expands. Set the top-level `image_size` larger than the base
  image; shrinking requires building a new image.
* Exactly one `ext2`, `ext3`, or `ext4` partition must have `size` set to `0`.
  Otherwise the builder cannot select the partition to expand.
* For "No space left on device", check `df -h` and `df -i`. Increase
  `image_size` only when the selected filesystem is out of blocks.

## Export as Docker image
With the `artifice` plugin you can pass a rootfs archive to docker post-processors
```json
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
```bash
cat .github/workflows/archlinuxarm-armv7-docker.yml
```

## How is this plugin different from `solo-io/packer-builder-arm-image`
https://github.com/hashicorp/packer/pull/8462

# Examples
For more examples please see:
```bash
tree boards/
```

The repository also includes some arm typical scripts to e.g. resize partitions on first boot or more extensive
provision scripts:

```bash
tree scripts/
```

A big resource for packer provisions scripts is the [GitHub Actions runner images](https://github.com/actions/runner-images) repository.

# Troubleshooting
Many of the reported issues are platform/OS specific. If you happen to have
problems, the first question you should ask yourself is:
> Is my setup faulty? or is there an actual issue?

To answer that question, I'd recommend reproducing the error on the VM, for
instance:
```bash
cd packer-plugin-builder-arm
vagrant up
vagrant provision
```
> Note: For this the disksize plugin is needed if not already installed `vagrant plugin install vagrant-disksize`

## Debugging builds
Enable verbose logging with:
```bash
PACKER_LOG=1 packer build ...
```

This builder does not pause for `packer build -debug`. Add a breakpoint before
the suspect provisioner instead:

```hcl
provisioner "breakpoint" {
  note = "Inspect the mounted image, then press Enter to continue"
}
```

While paused, find the root mountpoint in the log and inspect it from another
terminal:

```bash
sudo chroot /tmp/<mountpoint> /bin/bash
```

Run the suspect commands there.

## "Failed to find binfmt_misc for qemu-arm under /proc/sys/fs/binfmt_misc"
The plugin needs the qemu user-mode emulators registered with the kernel's
`binfmt_misc` mechanism on the host. Install both QEMU and its registration
package where available:

* Older Debian/Ubuntu:
  `sudo apt install qemu-user-static binfmt-support`
* Ubuntu 26.04 and other releases with unsuffixed binaries:
  `sudo apt install qemu-user qemu-user-binfmt`
* Arch Linux: install `qemu-user-static` **and** `qemu-user-static-binfmt`
* Fedora: `sudo dnf install qemu-user-static qemu-user-binfmt`, then run
  `sudo systemctl restart systemd-binfmt.service`
* RHEL/Rocky 8: the official repositories do not provide those QEMU user-mode
  packages. Use another trusted source, register QEMU manually, or use this
  project's container.

This project's privileged container registers its bundled interpreters, so it
needs no separate host setup. Check a registration with
`cat /proc/sys/fs/binfmt_misc/qemu-arm`. `qemu_binary_source_path` must resolve
to its interpreter file, such as `/usr/bin/qemu-arm-static` or
`/usr/bin/qemu-arm`; symlinks work.

# Demo
[![asciicast](https://asciinema.org/a/7ad1nm2Q7DRFVlHpqAknPolNo.svg)](https://asciinema.org/a/7ad1nm2Q7DRFVlHpqAknPolNo)
