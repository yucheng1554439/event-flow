variable "cluster_name" {
  type = string
}

variable "subnet_ids" {
  type = list(string)
}

variable "vpc_id" {
  type = string
}

resource "aws_msk_cluster" "kafka" {
  cluster_name           = var.cluster_name
  kafka_version          = "3.5.1"
  number_of_broker_nodes = 3

  broker_node_group_info {
    instance_type  = "kafka.m5.large"
    client_subnets = var.subnet_ids
    storage_info {
      ebs_storage_info { volume_size = 100 }
    }
    security_groups = [aws_security_group.msk.id]
  }

  encryption_info {
    encryption_in_transit {
      client_broker = "TLS"
      in_cluster    = true
    }
  }
}

resource "aws_security_group" "msk" {
  name   = "${var.cluster_name}-msk-sg"
  vpc_id = var.vpc_id
  ingress {
    from_port   = 9092
    to_port     = 9098
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

output "bootstrap_brokers" { value = aws_msk_cluster.kafka.bootstrap_brokers_tls }
