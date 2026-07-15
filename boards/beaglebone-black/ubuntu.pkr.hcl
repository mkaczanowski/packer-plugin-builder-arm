packer {
  required_plugins {
    arm = {
      version = ">= 1.0.9"
      source  = "github.com/mkaczanowski/arm"
    }
  }
}

source "arm" "ubuntu" {
  file_checksum         = "378adf5394ca0ae1e24ba434205ae93d579b29fe97da044905f6935ff9834098"
  file_checksum_type    = "sha256"
  file_target_extension = "xz"
  file_unarchive_cmd    = ["xz", "-d", "$ARCHIVE_PATH"]
  file_urls             = ["https://rcn-ee.net/rootfs/ubuntu-armhf-20.04-console-v5.10-ti/2023-12-21/am335x-ubuntu-20.04.6-console-armhf-2023-12-21-4gb.img.xz"]
  image_build_method    = "resize"
  image_path            = "bbb-sdcard-ubuntu-22.04.5-console.img"
  image_size            = "6G"
  image_type            = "dos"
  image_partitions {
    filesystem   = "ext4"
    mountpoint   = "/"
    name         = "root"
    size         = "0"
    start_sector = "8192"
    type         = "83"
  }
  image_chroot_env             = ["PATH=/usr/local/bin:/usr/local/sbin:/usr/bin:/bin:/sbin:/usr/sbin"]
  qemu_binary_destination_path = "/usr/bin/qemu-arm-static"
  qemu_binary_source_path      = "/usr/bin/qemu-arm-static"
}


build {
  sources = ["source.arm.ubuntu"]

  provisioner "shell" {
    inline = [
      "rm -f /etc/resolv.conf",
      "echo 'nameserver 1.1.1.1' > /etc/resolv.conf",
      "echo 'nameserver 8.8.8.8' >> /etc/resolv.conf",
      "apt-get update",
      "apt upgrade --yes --option=Dpkg::Options::=--force-confdef",
      "apt-get --yes autoremove",
      "apt-get --yes clean"
    ]
  }

}
