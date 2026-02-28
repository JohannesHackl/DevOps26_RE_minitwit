{ config, pkgs, lib, ... }:
let
  minitwit = pkgs.buildGoModule {
    pname = "minitwit";
    version = "latest";
    src = ./.;
    vendorHash = null;
  };
in
{
  options.services.minitwit = {
    dbAddr = lib.mkOption {
      type = lib.types.str;
      description = "Database server IP address";
    };
  };

  config = {
    systemd.services.minitwit = {
      description = "Minitwit go application";
      wantedBy = [ "multi-user.target" ];
      after = [ "network.target" ];
      environment = {
        DB_ADDR = config.services.minitwit.dbAddr;
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
    users.groups.minitwit = { };

    networking.firewall.allowedTCPPorts = [ 5001 ];
  };
}
