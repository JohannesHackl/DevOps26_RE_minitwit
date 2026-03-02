# -*- mode: ruby -*-
# vi: set ft=ruby :

#$ip_file = "db_ip.txt"

Vagrant.configure("2") do |config|
  config.vm.box = 'digital_ocean'
  config.vm.box_url = "https://github.com/devopsgroup-io/vagrant-digitalocean/raw/master/box/digital_ocean.box"
  config.ssh.private_key_path = '~/.ssh/id_rsa'
  # Sync project folder
  config.vm.synced_folder ".", "/vagrant", type: "rsync"

  # --- Database Server with Docker ---
  config.vm.define "dbserver" do |db|
    db.vm.provider :digital_ocean do |provider|
      provider.ssh_key_name = ENV["SSH_KEY_NAME"]
      provider.token        = ENV["DIGITAL_OCEAN_TOKEN"]
      provider.image        = 'ubuntu-22-04-x64'
      provider.region       = 'fra1'
      provider.size         = 's-1vcpu-1gb'
      provider.privatenetworking = true
    end

    db.vm.hostname = "dbserver"

    db.vm.provision "ansible" do |ansible|
      ansible.playbook = "ansible/site.yml"
      #ansible.inventory_path = "ansible/inventory.ini"
      ansible.limit = "dbserver"
      ansible.verbose = "v"
    end
  end 

  # --- Web Server with Doscker ---
  config.vm.define "webserver" do |web|
    
    web.vm.provider :digital_ocean do |provider|
      provider.ssh_key_name = ENV["SSH_KEY_NAME"]
      provider.token        = ENV["DIGITAL_OCEAN_TOKEN"]
      provider.image        = 'ubuntu-22-04-x64'
      provider.region       = 'fra1'
      provider.size         = 's-1vcpu-1gb'
      provider.privatenetworking = true
    end

    web.vm.hostname = "webserver"

    web.vm.provision "ansible" do |ansible|
      ansible.playbook = "ansible/site.yml"
      #ansible.inventory_path = "ansible/inventory.ini"
      ansible.limit = "webserver"
      ansible.verbose = "v"
    end
  end
end
