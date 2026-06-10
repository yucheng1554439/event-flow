variable "cluster_id" {
  type = string
}

variable "subnet_ids" {
  type = list(string)
}

variable "vpc_id" {
  type = string
}

resource "aws_elasticache_subnet_group" "main" {
  name       = "${var.cluster_id}-subnet"
  subnet_ids = var.subnet_ids
}

resource "aws_security_group" "redis" {
  name   = "${var.cluster_id}-redis-sg"
  vpc_id = var.vpc_id
  ingress {
    from_port   = 6379
    to_port     = 6379
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }
}

resource "aws_elasticache_replication_group" "redis" {
  replication_group_id       = var.cluster_id
  description                = "EventFlow Redis cache"
  node_type                  = "cache.r6g.large"
  num_cache_clusters         = 2
  automatic_failover_enabled = true
  subnet_group_name          = aws_elasticache_subnet_group.main.name
  security_group_ids         = [aws_security_group.redis.id]
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
}

output "primary_endpoint" { value = aws_elasticache_replication_group.redis.primary_endpoint_address }
