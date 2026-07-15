integration {
  name        = "Arm"
  description = "The Arm builder plugin can be used with HashiCorp Packer to create ARM system images (new, reuse or resize) via qemu/chroot."
  identifier  = "packer/mkaczanowski/arm"
  flags       = []

  component {
    type = "builder"
    name = "Arm"
    slug = "arm"
  }
}
