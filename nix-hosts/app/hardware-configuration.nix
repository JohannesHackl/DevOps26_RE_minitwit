{ config, lib, pkgs, ... }:
{
  fileSystems."/" = {
    device = "/dev/sda1";
    fsType = "ext4";
  };

  boot.loader.grub = {
    enable = true;
    device = "/dev/sda";
  };
}
