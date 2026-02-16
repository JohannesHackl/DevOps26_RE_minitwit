# -*- mode: ruby -*-
# vi: set ft=ruby :

$ip_file = "db_ip.txt"

Vagrant.configure("2") do |config|
  config.vm.box = 'digital_ocean'
  config.vm.box_url = "https://github.com/devopsgroup-io/vagrant-digitalocean/raw/master/box/digital_ocean.box"
  config.ssh.private_key_path = '~/.ssh/id_rsa'
  config.vm.synced_folder ".", "/vagrant", type: "rsync"

  config.vm.define "dbserver", primary: true do |server|
    server.vm.provider :digital_ocean do |provider|
      provider.ssh_key_name = ENV["SSH_KEY_NAME"]
      provider.token = ENV["DIGITAL_OCEAN_TOKEN"]
      provider.image = 'ubuntu-22-04-x64'
      provider.region = 'fra1'
      provider.size = 's-1vcpu-1gb'
      provider.privatenetworking = true
    end

    server.vm.hostname = "dbserver"

    server.trigger.after :up do |trigger|
      trigger.info =  "Writing dbserver's IP to file..."
      trigger.ruby do |env,machine|
        remote_ip = machine.instance_variable_get(:@communicator).instance_variable_get(:@connection_ssh_info)[:host]
        File.write($ip_file, remote_ip)
      end
    end

    server.vm.provision "shell", inline: <<-SHELL
      # The following addresses an issue in DO's Ubuntu images, which still contain a lock file
      sudo fuser -vk -TERM /var/lib/apt/lists/lock
      sudo apt-get update

      # See installation instructions: https://www.mongodb.com/docs/v8.0/tutorial/install-mongodb-on-ubuntu/
      sudo apt-get install -y gnupg curl
      echo "Installing MongoDB"
      curl -fsSL https://www.mongodb.org/static/pgp/server-8.0.asc | sudo gpg -o /usr/share/keyrings/mongodb-server-8.0.gpg --dearmor
      echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-8.0.gpg ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/8.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-8.0.list

      sudo apt-get update
      sudo apt-get install -y mongodb-org

      sudo mkdir -p /data/db
      sudo sed -i '/  bindIp:/ s/127.0.0.1/0.0.0.0/' /etc/mongod.conf

      sudo systemctl start mongod
      mongorestore --gzip /vagrant/dump
    SHELL
  end

  config.vm.define "webserver", primary: false do |server|

    server.vm.provider :digital_ocean do |provider|
      provider.ssh_key_name = ENV["SSH_KEY_NAME"]
      provider.token = ENV["DIGITAL_OCEAN_TOKEN"]
      provider.image = 'ubuntu-22-04-x64'
      provider.region = 'fra1'
      provider.size = 's-1vcpu-1gb'
      provider.privatenetworking = true
    end

    server.vm.hostname = "webserver"

    server.trigger.before :up do |trigger|
      trigger.info =  "Waiting to create server until dbserver's IP is available."
      trigger.ruby do |env,machine|
        ip_file = "db_ip.txt"
        while !File.file?($ip_file) do
          sleep(1)
        end
        db_ip = File.read($ip_file).strip()
        puts "Now, I have the dbserver's IP address: #{db_ip}"
      end
    end

    server.trigger.before :destroy do |trigger|
      trigger.info =  "Cleaning dbserver's IP file."
      trigger.ruby do |env,machine|
        File.delete($ip_file) if File.exist? $ip_file
      end
    end

    server.vm.provision "shell", inline: <<-SHELL
      # The following addresses an issue in DO's Ubuntu images, which still contain a lock file
      sudo fuser -vk -TERM /var/lib/apt/lists/lock
      sudo apt-get update

      export DB_IP=$(cat /vagrant/db_ip.txt)
      echo $DB_IP

      sudo apt-get install -y python-is-python3 python3-pip
      pip install Flask-PyMongo

      cp -r /vagrant/* $HOME
      nohup python minitwit.py > out.log &
      echo "================================================================="
      echo "=                            DONE                               ="
      echo "================================================================="
      echo "Navigate in your browser to:"
      THIS_IP=`hostname -I | cut -d" " -f1`
      echo "http://${THIS_IP}:5000"
    SHELL
  end
end