# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  # https://github.com/gusztavvargadr/packer/
  config.vm.box = "gusztavvargadr/windows-server-2019-standard"
  config.vm.provision "shell", path: "scripts/provision-windows-box.ps1"
end
