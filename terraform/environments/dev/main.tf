terraform {
  required_version = ">= 1.6"
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 5.0" }
  }
}

provider "aws" {
  region = var.aws_region
}

module "vpc" {
  source = "../../modules/vpc"
  name   = "eventflow-${var.environment}"
}

module "eks" {
  source       = "../../modules/eks"
  cluster_name = "eventflow-${var.environment}"
  vpc_id       = module.vpc.vpc_id
  subnet_ids   = module.vpc.private_subnet_ids
}

module "msk" {
  source       = "../../modules/msk"
  cluster_name = "eventflow-kafka-${var.environment}"
  vpc_id       = module.vpc.vpc_id
  subnet_ids   = module.vpc.private_subnet_ids
}

module "rds" {
  source     = "../../modules/rds"
  identifier = "eventflow-${var.environment}"
  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnet_ids
  password   = var.db_password
}

module "elasticache" {
  source     = "../../modules/elasticache"
  cluster_id = "eventflow-redis-${var.environment}"
  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnet_ids
}

output "eks_cluster" { value = module.eks.cluster_endpoint }
output "kafka_brokers" { value = module.msk.bootstrap_brokers }
output "postgres_endpoint" { value = module.rds.endpoint }
output "redis_endpoint" { value = module.elasticache.primary_endpoint }
