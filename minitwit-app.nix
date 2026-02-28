{ config, pkgs, lib, ... }:
let
  minitwit = pkgs.buildGoModule {
    pname = "minitwit";
    version = "latest";
    src = fetchFromGitHub {
      owner = "JohannesHackl";
      repo = "DevOps26_RE_minitwit";
      rev = "main";
      hash = lib.fakeSha256;
    };
    vendorHash = null;
  };
in
{
  systemd.services.minitwit = {
    description = "Minitwit go application";
    wantedBy = [ "multi-user.target" ];
    after = [ "network.target" ];
    environment = {
      DB_ADDR = "";
    };
    serviceConfig = {
      ExecStart = "${minitwit}/bin/minitwit";
      WorkingDirectory = "/var/lib/minitwit";
      Restart = "always";
      User = "minitwit";
    };
  };

  systemd.tmpfiles.rules = [
    "d /var/lib/minitwit 0750 minitwit minitwit -"
    "L /var/lib/minitwit/templates - - - - ${minitwit}/share/minitwit/templates"
    "L /var/lib/minitwit/static - - - - ${minitwit}/share/minitwit/static"
  ];

  users.users.mintwit = {
    isSystemUser = true;
    group = "minitwit";
  };
  users.group.minitwit = { };

  networking.firewall.allowedTCPPorts = [ 5001 ];
}
