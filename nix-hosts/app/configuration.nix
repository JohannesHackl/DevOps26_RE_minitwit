{ config, pkgs, ... }:
{

  imports = [
    ../../modules/minitwit-app.nix
    ./hardware-configuration.nix
  ];

  services.minitwit-app.dbAddr = "164.92.186.201";

  networking.hostName = "minitwit-app";

  services.openssh.enable = true;
  users.users.root.openssh.authorizedKeys.keys = [
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJrmlWbyrXyqEI8nP/N31d1yfT314rk3Jr7DS47f6Q27 desktop ssh"
  ];

  system.stateVersion = "25.05";
}
