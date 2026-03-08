terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
  }
}

variable "do_token" {
  description = "DigitalOcean API token"
  type        = string
  sensitive   = true
}

variable "do_region" {
  description = "DigitalOcean region slug"
  type        = string
  default     = "nyc3"
}

variable "namespace" {
  description = "Chain namespace (derived from chain name)"
  type        = string
}

provider "digitalocean" {
  token = var.do_token
}

# DO Spaces bucket for Terraform state
resource "digitalocean_spaces_bucket" "terraform_state" {
  name   = "trh-terraform-state-${var.namespace}"
  region = var.do_region

  lifecycle_rule {
    enabled = true
    expiration {
      days = 365
    }
  }
}

output "spaces_bucket_name" {
  value = digitalocean_spaces_bucket.terraform_state.name
}

output "spaces_region" {
  value = digitalocean_spaces_bucket.terraform_state.region
}
