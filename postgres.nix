{ config, pkgs, ... }:
{
  services.postgresql = {
    enable = true;
    package = pkgs.postgresql_14;
    enableTCPIP = true;
    authentication = ''
      host all all 0.0.0.0/0 md4
    '';
    ensureDatabases = [ "minitwit" ];
    ensureUsers = [{
      name = "minitwit";
      ensureDBOwnership = true;
    }];
    initialScript = pkgs.writeText "init.sql" ''
      ALTER USER minitwit WITH PASSWORD 'minitwit';
    '';

  };
  networking.firewall.allowedTCPPorts = [ 5432 ];
}
