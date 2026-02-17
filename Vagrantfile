# -*- mode: ruby -*-
# vi: set ft=ruby :

$ip_file = "db_ip.txt"

Vagrant.configure("2") do |config|
  config.vm.box = 'digital_ocean'
  config.vm.box_url = "https://github.com/devopsgroup-io/vagrant-digitalocean/raw/master/box/digital_ocean.box"
  config.ssh.private_key_path = '~/.ssh/id_rsa'
  config.vm.synced_folder ".", "/vagrant", type: "rsync"

  # --- Database Server ---
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

      # Install PostgreSQL
      sudo apt-get update
      sudo apt-get install -y postgresql postgresql-contrib

      # Configure PostgreSQL to listen on all interfaces
      # By default it only listens on localhost (127.0.0.1)
      sudo sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" \
        /etc/postgresql/14/main/postgresql.conf

      # Allow connections from md5
      # pg_hba.conf controls WHO can connect and HOW they authenticate
      echo "host all all 0.0.0.0/0 md5" | sudo tee -a /etc/postgresql/14/main/pg_hba.conf

      sudo systemctl restart postgresql

      # Create database and user
      sudo -u postgres psql -c "CREATE USER minitwit WITH PASSWORD 'minitwit';"
      sudo -u postgres psql -c "CREATE DATABASE minitwit OWNER minitwit;"

      # Load the schema as give permission to minitwit.
      PGPASSWORD=minitwit psql -h 127.0.0.1 -U minitwit -d minitwit -f /vagrant/src/bin/schema.sql
    SHELL
  end

  # --- Web Server ---
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

    web.vm.provision "shell", inline: <<-SHELL
      export DEBIAN_FRONTEND=noninteractive

      echo "Waiting for apt lock to be released..."
      sudo fuser -vk -TERM /var/lib/apt/lists/lock || true
      sudo fuser -vk -TERM /var/lib/dpkg/lock-frontend || true
      sudo dpkg --configure -a

      # Install Go
      sudo apt-get update
      sudo apt-get install -y wget

      # Create a swap in case the memory not enough
      if [ ! -f "/swapfile" ]; then
        echo "Creating 2GB Swap file for Go compilation..."
        sudo fallocate -l 2G /swapfile
        sudo chmod 600 /swapfile
        sudo mkswap /swapfile
        sudo swapon /swapfile
      fi

      if [ ! -d "/usr/local/go" ]; then
        wget -q https://dl.google.com/go/go1.25.0.linux-amd64.tar.gz
        sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
        rm go1.25.0.linux-amd64.tar.gz
      fi

      # Write to persistent path
      echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
      echo 'export GOPATH=/home/vagrant/go' | sudo tee -a /etc/profile.d/go.sh
      source /etc/profile.d/go.sh

      cd /vagrant/src
      echo "Cleaning old binaries..."
      rm -rf ./bin/minitwit

      # Write DB IP Value to environment variable
      DB_IP_VALUE=$(cat /vagrant/db_ip.txt)
      echo "export DB_ADDR=$DB_IP_VALUE" | sudo tee /etc/profile.d/db_env.sh

      if ! grep -q "DB_ADDR=" /etc/environment; then
        echo "DB_ADDR=$DB_IP_VALUE" | sudo tee -a /etc/environment
      else
        sudo sed -i "s|DB_ADDR=.*|DB_ADDR=$DB_IP_VALUE|" /etc/environment
      fi

      export DB_ADDR=$DB_IP_VALUE
      echo "Connecting to Database at: $DB_ADDR"

      # Build the application
      cd /vagrant/src
      echo "Downloading Go modules..."
      /usr/local/go/bin/go mod download

      echo "Building application..."
      /usr/local/go/bin/go build -o ./bin/minitwit ./...

      echo "Stopping any existing minitwit processes..."
      sudo pkill minitwit || true

      echo "Starting Minitwit in background..."
      cd bin
      # Run the app in the background, logging all output and ensuring it persists after logout.
      nohup ./minitwit > minitwit.log 2>&1 &

      echo "=========================================================="
      echo "Deployment Complete!"
      THIS_IP=$(curl -s http://checkip.amazonaws.com)
      echo "Access your app at http://$THIS_IP:5001"
      echo "=========================================================="
    SHELL
  end
end
