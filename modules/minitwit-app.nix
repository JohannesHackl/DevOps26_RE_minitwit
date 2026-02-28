{ config, pkgs, lib, ... }:
let
  minitwit = pkgs.buildGoModule {
    pname = "minitwit";
    version = "latest";
    src = ../.;
    vendorHash = "sha256-zVj7biULqStZsrAe3xLNkOOX3ol/RLMUitmd2YujSLM=";
    ###    postInstall = ''
    ###      mkdir -p $out/share/minitwit
    ###      cp -r templates $out/share/minitwit/templates
    ###      cp -r static $out/share/minitwit/static
    ###    '';
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
        TEMPLATES_PATH = "${minitwit}/share/minitwit/templates";
        STATIC_PATH = "${minitwit}/share/minitwit/static";
      };
      serviceConfig = {
        ExecStart = "${minitwit}/bin/RE_minitwit";
        Restart = "always";
        User = "minitwit";
      };
    };

    ###    systemd.tmpfiles.rules = [
    ###      "d /var/lib/minitwit 0750 minitwit minitwit -"
    ###      "L /var/lib/minitwit/templates - - - - ${minitwit}/share/minitwit/templates"
    ###      "L /var/lib/minitwit/static - - - - ${minitwit}/share/minitwit/static"
    ###    ];

    users.users.minitwit = {
      isSystemUser = true;
      group = "minitwit";
    };
    users.groups.minitwit = { };

    networking.firewall.allowedTCPPorts = [ 5001 ];
  };
}
