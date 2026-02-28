variable "ssh_key_id" {
  description = "DigitalOcean SSH key ID"
  type        = string
}

variable "do_token" {
  description = "DigitalOcean API token"
  type        = string
  sensitive   = true
}
