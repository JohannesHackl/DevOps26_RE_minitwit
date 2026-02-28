{ config, pkgs, ... }:
{

  imports = [
    ../../modules/minitwit-app.nix
    ./hardware-configuration.nix
  ];

  services.minitwit.dbAddr = "164.92.186.201";

  networking.hostName = "minitwit-app";

  services.openssh = {
    enable = true;
    settings.PermitRootLogin = "yes";
  };
  users.users.root.openssh.authorizedKeys.keys = [
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIJrmlWbyrXyqEI8nP/N31d1yfT314rk3Jr7DS47f6Q27 desktop ssh"
  ];

  users.users.root.password = "123";

  virtualisation.vmVariant = {
    virtualisation.forwardPorts = [
      { from = "host"; host.port = 2222; guest.port = 22; }
    ];
  };

  system.stateVersion = "25.05";
}
