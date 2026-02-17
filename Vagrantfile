# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "bento/ubuntu-22.04"

  config.vm.provider "virtualbox" do |vb|
    vb.memory = "1024"
  end

  # --- Database Server ---
  config.vm.define "dbserver" do |db|
    db.vm.hostname = "dbserver"
    db.vm.network "private_network", ip: "192.168.56.10"

    db.vm.provision "shell", inline: <<-SHELL
      export DEBIAN_FRONTEND=noninteractive

      # Install PostgreSQL
      apt-get update
      apt-get install -y postgresql postgresql-contrib

      # Configure PostgreSQL to listen on all interfaces
      # By default it only listens on localhost (127.0.0.1)
      sed -i "s/#listen_addresses = 'localhost'/listen_addresses = '*'/" \
        /etc/postgresql/14/main/postgresql.conf

      # Allow connections from the private network
      # pg_hba.conf controls WHO can connect and HOW they authenticate
      echo "host all all 192.168.56.0/24 md5" >> \
        /etc/postgresql/14/main/pg_hba.conf

      systemctl restart postgresql

      # Create database and user
      sudo -u postgres psql -c "CREATE USER minitwit WITH PASSWORD 'minitwit';"
      sudo -u postgres psql -c "CREATE DATABASE minitwit OWNER minitwit;"

      # Load the schema as give permission to minitwit.
      PGPASSWORD=minitwit psql -h 127.0.0.1 -U minitwit -d minitwit -f /vagrant/src/bin/schema.sql
    SHELL
  end

  # --- Web Server ---
  config.vm.define "webserver" do |web|
    web.vm.hostname = "webserver"
    web.vm.network "private_network", ip: "192.168.56.11"
    web.vm.network "forwarded_port", guest: 5001, host: 5001

    web.vm.provision "shell", inline: <<-SHELL
      export DEBIAN_FRONTEND=noninteractive

      # Install Go
      apt-get update
      apt-get install -y wget
      wget -q https://dl.google.com/go/go1.25.0.linux-amd64.tar.gz
      tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
      rm go1.25.0.linux-amd64.tar.gz

      echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh
      echo 'export GOPATH=/home/vagrant/go' >> /etc/profile.d/go.sh
      source /etc/profile.d/go.sh

      # Build the application
      cd /vagrant/src
      echo "Downloading Minitwit modules..."
      go mod download
      echo "Building Minitwit application..."
      go build -o ./bin/minitwit ./...
      echo "Build Complete!"
      echo "Running Minitwit..."
      cd bin
      ./minitwit
    SHELL
  end
end
