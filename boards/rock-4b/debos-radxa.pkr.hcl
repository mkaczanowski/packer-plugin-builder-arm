packer {
  required_plugins {
    arm = {
      version = ">= 1.0.9"
      source  = "github.com/mkaczanowski/arm"
    }
  }
}

source "arm" "radxa" {
  file_urls             = ["https://github.com/radxa/debos-radxa/releases/download/20221201-0835/rockpi-4b-debian-bullseye-xfce4-arm64-20221201-1203-gpt.img.xz"]
  file_checksum         = "14d5c5318e5a5bd407e718ba01a03a4e631895007fb6ddde1597310d23782894"
  file_checksum_type    = "sha256"
  file_target_extension = "xz"
  file_unarchive_cmd    = ["xz", "--decompress", "$ARCHIVE_PATH"]
  image_build_method    = "reuse"
  image_path            = "rock-4b-radxa-debian11.img"
  image_size            = "3.8G"
  image_type            = "gpt"
  image_partitions {
    name         = ""
    type         = "c"
    start_sector = "64"
    filesystem   = "fat"
    size         = "3.9M"
    mountpoint   = ""
  }
  image_partitions {
    name         = ""
    type         = "c"
    start_sector = "16384"
    filesystem   = "fat"
    size         = "4M"
    mountpoint   = ""
  }
  image_partitions {
    name         = ""
    type         = "c"
    start_sector = "24576"
    filesystem   = "fat"
    size         = "4M"
    mountpoint   = ""
  }
  image_partitions {
    name         = "boot"
    type         = "c"
    start_sector = "32768"
    filesystem   = "fat"
    size         = "512M"
    mountpoint   = "/boot"
  }
  image_partitions {
    name         = "root"
    type         = "83"
    start_sector = "1081344"
    filesystem   = "ext4"
    size         = "3.2G"
    mountpoint   = "/"
  }
  image_chroot_env             = ["PATH=/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin"]
  qemu_binary_source_path      = "/usr/bin/qemu-aarch64-static"
  qemu_binary_destination_path = "/usr/bin/qemu-aarch64-static"
}

build {
  sources = ["source.arm.radxa"]

  provisioner "shell" {
    inline = [
      "touch /tmp/test",
    ]
  }
}
