terraform {
  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.0"
    }
  }

  # Backend flags are injected at runtime by deploy_chain.go STEP 5 (AWS_ACCESS_KEY_ID = DO Spaces key).
  backend "s3" {}
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
  description = "Chain namespace"
  type        = string
}

variable "sequencer_key" {
  description = "Sequencer private key"
  type        = string
  sensitive   = true
}

variable "batcher_key" {
  description = "Batcher private key"
  type        = string
  sensitive   = true
}

variable "proposer_key" {
  description = "Proposer private key"
  type        = string
  sensitive   = true
}

variable "challenger_key" {
  description = "Challenger private key"
  type        = string
  sensitive   = true
  default     = ""
}

variable "l1_rpc_url" {
  description = "L1 RPC URL"
  type        = string
}

variable "l1_beacon_url" {
  description = "L1 Beacon URL"
  type        = string
}

variable "thanos_stack_image_tag" {
  description = "Thanos stack Docker image tag"
  type        = string
}

variable "op_geth_image_tag" {
  description = "op-geth Docker image tag"
  type        = string
}

variable "node_size" {
  description = "DOKS node size"
  type        = string
  default     = "s-4vcpu-8gb"
}

variable "node_count" {
  description = "DOKS node count"
  type        = number
  default     = 2
}

variable "k8s_version" {
  description = "DOKS Kubernetes version slug (e.g. 1.30.x-do.0)"
  type        = string
  default     = "1.30.x-do.0"
}

variable "vpc_cidr" {
  description = "VPC IP range in CIDR notation"
  type        = string
  default     = "10.10.0.0/16"
}

variable "db_size" {
  description = "Managed PostgreSQL cluster size slug"
  type        = string
  default     = "db-s-1vcpu-1gb"
}

provider "digitalocean" {
  token = var.do_token
}

# VPC
resource "digitalocean_vpc" "thanos_vpc" {
  name     = "${var.namespace}-vpc"
  region   = var.do_region
  ip_range = var.vpc_cidr
}

# DOKS cluster
resource "digitalocean_kubernetes_cluster" "thanos_cluster" {
  name     = var.namespace
  region   = var.do_region
  version  = var.k8s_version
  vpc_uuid = digitalocean_vpc.thanos_vpc.id

  node_pool {
    name       = "${var.namespace}-worker-pool"
    size       = var.node_size
    node_count = var.node_count

    labels = {
      chain = var.namespace
    }
  }
}

# Managed PostgreSQL (for block explorer)
resource "digitalocean_database_cluster" "blockscout_db" {
  name       = "${var.namespace}-db"
  engine     = "pg"
  version    = "15"
  size       = var.db_size
  region     = var.do_region
  node_count = 1

  private_network_uuid = digitalocean_vpc.thanos_vpc.id
}

# Kubernetes secrets for private keys
resource "kubernetes_secret" "chain_keys" {
  depends_on = [digitalocean_kubernetes_cluster.thanos_cluster]

  metadata {
    name      = "${var.namespace}-keys"
    namespace = "default"
  }

  data = {
    sequencer_key  = var.sequencer_key
    batcher_key    = var.batcher_key
    proposer_key   = var.proposer_key
    challenger_key = var.challenger_key
  }
}

output "cluster_id" {
  value = digitalocean_kubernetes_cluster.thanos_cluster.id
}

output "cluster_endpoint" {
  value = digitalocean_kubernetes_cluster.thanos_cluster.endpoint
}

output "database_host" {
  value     = digitalocean_database_cluster.blockscout_db.private_host
  sensitive = true
}

output "database_port" {
  value = digitalocean_database_cluster.blockscout_db.port
}
