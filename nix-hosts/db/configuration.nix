{ config, pkgs, lib, ... }:
{
  imports = [ ../../modules/postgres.nix ];

  networking.hostNames = "minitwit-db";
  services.openssh.enable = true;
  users.users.root.openssh.authorizedKeys.keys = [
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJrmlWbyrXyqEI8nP/N31d1yfT314rk3Jr7DS47f6Q27 desktop ssh"
  ];

  system.stateVersion = "25.05";
}
