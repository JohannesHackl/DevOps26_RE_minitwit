terraform {
  required_providers {
    digitalocean = {
      source = "digitalocean/digitalocean"
    }
  }
}

provider "digitalocean" {
  token = var.do_token
}
resource "digitalocean_droplet" "db" {
  name   = "minitwit-db"
  image  = "ubuntu-22-04-x64"
  size   = "s-1vcpu-1gb"
  region = "fra1"
  ssh_keys = [var.ssh_key_id]
}

resource "digitalocean_droplet" "web" {
  name   = "minitwit-web"
  image  = "ubuntu-22-04-x64"
  size   = "s-1vcpu-1gb"
  region = "fra1"
  ssh_keys = [var.ssh_key_id]
}

output "db_private_ip" {
  value = digitalocean_droplet.db.ipv4_address_private
}

output "web_public_ip" {
  value = digitalocean_droplet.web.ipv4_address
}
