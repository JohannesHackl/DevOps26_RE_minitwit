# -*- mode: ruby -*-
# vi: set ft=ruby :

$ip_file = "db_ip.txt"

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
  config.vm.synced_folder ".", "/vagrant", type: "rsync"

  # --- Database Server with Docker ---
  config.vm.define "dbserver" do |db|
    db.vm.provider :digital_ocean do |provider|
      provider.ssh_key_name = ENV["SSH_KEY_NAME"]
      provider.token = ENV["DIGITAL_OCEAN_TOKEN"]
      provider.image = 'ubuntu-22-04-x64'
      provider.region = 'fra1'
      provider.size = 's-1vcpu-1gb'
      provider.privatenetworking = true
    end

    db.vm.hostname = "dbserver"

    db.trigger.after :up do |trigger|
      trigger.info = "Writing dbserver's IP to file..."
      trigger.ruby do |env, machine|
        remote_ip = machine.instance_variable_get(:@communicator).instance_variable_get(:@connection_ssh_info)[:host]
        File.write($ip_file, remote_ip)
      end
    end

    db.vm.provision "shell", inline: <<-SHELL
      export DEBIAN_FRONTEND=noninteractive

      echo "Installing Docker (fast method)..."
      curl -fsSL https://get.docker.com | sudo sh

      echo "Starting Docker service..."
      sudo systemctl start docker
      sudo systemctl enable docker

      cd /vagrant
      mkdir -p tmp

      echo "Starting PostgreSQL container..."
      sudo docker compose -f docker-compose-db.yaml up -d

      echo "=========================================================="
      echo "Database Server Ready!"
      echo "PostgreSQL accessible at: $(hostname -I | awk '{print $1}'):5432"
      echo "=========================================================="
    SHELL
  end

  # --- Web Server with Docker ---
  config.vm.define "webserver" do |web|
    web.vm.provider :digital_ocean do |provider|
      provider.ssh_key_name = ENV["SSH_KEY_NAME"]
      provider.token = ENV["DIGITAL_OCEAN_TOKEN"]
      provider.image = 'ubuntu-22-04-x64'
      provider.region = 'fra1'
      provider.size = 's-1vcpu-1gb'
      provider.privatenetworking = true
    end

    web.vm.hostname = "webserver"

    web.trigger.before :up do |trigger|
      trigger.info = "Waiting for dbserver's IP..."
      trigger.ruby do |env, machine|
        while !File.file?($ip_file) do
          sleep(1)
        end
        db_ip = File.read($ip_file).strip()
        puts "Database found at: #{db_ip}"
      end
    end


    web.vm.provision "ansible" do |ansible|
        ansible.playbook = "ansible/site.yml"
        ansible.inventory_path = "ansible/inventory.ini"
    end
  end
end