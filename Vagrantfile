# -*- mode: ruby -*-
# vi: set ft=ruby :

# All Vagrant configuration is done below. The "2" in Vagrant.configure
# configures the configuration version (we support older styles for
# backwards compatibility). Please don't change it unless you know what
# you're doing.
Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/jammy64"
  config.disksize.size = '40GB'

  config.vm.provision "shell", inline: <<-SHELL
    set -o errtrace -o nounset -o pipefail -o errexit
    curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -
    sudo apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"

    apt-get update
    apt-get install -y qemu-user-static packer libarchive-tools
    #apt-get install -y git golang
    #rm -rf packer-plugin-builder-arm *>/dev/null

    #git clone https://github.com/mkaczanowski/packer-plugin-builder-arm
    #cd packer-plugin-builder-arm
    #go mod download
    #go build

    #packer build boards/raspberry-pi-3/archlinuxarm.json
  SHELL
end
