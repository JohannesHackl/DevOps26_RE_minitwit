# -*- mode: ruby -*-
# vi: set ft=ruby :

def find_ssh_key
  ["~/.ssh/id_rsa", "~/.ssh/id_ed25519", "~/.ssh/id_dsa"].each do |path|
    full_path = File.expand_path(path)
    return path if File.exist?(full_path)
  end
  "~/.ssh/id_rsa"
end

ssh_key_path = ENV['SSH_KEY_PATH'] || find_ssh_key

Vagrant.configure("2") do |config|
  config.vm.box = 'digital_ocean'
  config.vm.box_url = "https://github.com/devopsgroup-io/vagrant-digitalocean/raw/master/box/digital_ocean.box"
  config.ssh.private_key_path = ssh_key_path
  config.vm.synced_folder ".", "/vagrant", type: "rsync",
    rsync__exclude: [".git/", ".venv/", "tmp/", "db_ip.txt"]

  # --- Single server running both db and web via docker-compose ---
  config.vm.define "webserver" do |web|
    web.vm.provider :digital_ocean do |provider|
      provider.ssh_key_name = ENV["SSH_KEY_NAME"]
      provider.token = ENV["DIGITAL_OCEAN_TOKEN"]
      provider.image = 'ubuntu-22-04-x64'
      provider.region = 'fra1'
      provider.size = 's-2vcpu-2gb'
      provider.privatenetworking = true
    end

    web.vm.hostname = "webserver"

    web.vm.provision "shell", inline: <<-SHELL
      export DEBIAN_FRONTEND=noninteractive

      echo "=== Installing Docker ==="
      curl -fsSL https://get.docker.com | sh
      sudo usermod -aG docker root

      echo "=== Starting app with docker compose ==="
      cd /vagrant
      docker compose up -d --build

      echo "=========================================================="
      echo "Deployment Complete! Access: http://$(curl -s http://checkip.amazonaws.com):5001"
      echo "=========================================================="
    SHELL
  end
end