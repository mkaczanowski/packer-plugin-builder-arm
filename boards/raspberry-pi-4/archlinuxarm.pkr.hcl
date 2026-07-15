packer {
  required_plugins {
    git = {
      version = ">=v0.3.2"
      source  = "github.com/ethanmdavidson/git"
    }
    arm = {
      version = ">= 1.0.9"
      source  = "github.com/mkaczanowski/arm"
    }
  }
}

source "arm" "arch" {
  file_urls             = ["http://os.archlinuxarm.org/os/ArchLinuxARM-rpi-aarch64-latest.tar.gz"]
  file_checksum_url     = "http://os.archlinuxarm.org/os/ArchLinuxARM-rpi-aarch64-latest.tar.gz.md5"
  file_checksum_type    = "md5"
  file_target_extension = "tar.gz"
  file_unarchive_cmd    = ["bsdtar", "-xpf", "$ARCHIVE_PATH", "-C", "$MOUNTPOINT"]
  image_build_method    = "new"
  image_partitions {
    filesystem   = "vfat"
    mountpoint   = "/boot"
    name         = "boot"
    size         = "256M"
    start_sector = "2048"
    type         = "c"
  }
  image_partitions {
    filesystem   = "ext4"
    mountpoint   = "/"
    name         = "root"
    size         = "0"
    start_sector = "526336"
    type         = "83"
  }
  image_path                   = "raspberry-pi-4.img"
  image_size                   = "2G"
  image_type                   = "dos"
  qemu_binary_destination_path = "/usr/bin/qemu-aarch64-static"
  qemu_binary_source_path      = "/usr/bin/qemu-aarch64-static"
}

build {
  sources = ["source.arm.arch"]

  provisioner "shell" {
    inline = [
      "mv /etc/resolv.conf /etc/resolv.conf.bk",
      "echo 'nameserver 8.8.8.8' > /etc/resolv.conf",
      "pacman-key --init",
      "pacman-key --populate archlinuxarm",
      "pacman -Sy --disable-sandbox --noconfirm --needed",
      "pacman -S --disable-sandbox parted --noconfirm --needed"
    ]
  }

  provisioner "file" {
    destination = "/tmp"
    source      = "scripts/resizerootfs"
  }

  provisioner "shell" {
    script = "scripts/bootstrap_resizerootfs.sh"
  }

}
