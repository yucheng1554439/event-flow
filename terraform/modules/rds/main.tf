variable "identifier" {
  type = string
}

variable "subnet_ids" {
  type = list(string)
}

variable "vpc_id" {
  type = string
}

variable "password" {
  type      = string
  sensitive = true
}

resource "aws_db_subnet_group" "main" {
  name       = "${var.identifier}-subnet"
  subnet_ids = var.subnet_ids
}

resource "aws_security_group" "rds" {
  name   = "${var.identifier}-rds-sg"
  vpc_id = var.vpc_id
  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/8"]
  }
}

resource "aws_db_instance" "postgres" {
  identifier              = var.identifier
  engine                  = "postgres"
  engine_version          = "16.1"
  instance_class          = "db.r6g.large"
  allocated_storage       = 100
  max_allocated_storage   = 500
  storage_encrypted       = true
  db_name                 = "eventflow"
  username                = "eventflow"
  password                = var.password
  db_subnet_group_name    = aws_db_subnet_group.main.name
  vpc_security_group_ids  = [aws_security_group.rds.id]
  multi_az                = true
  backup_retention_period = 7
  skip_final_snapshot     = true
}

output "endpoint" { value = aws_db_instance.postgres.endpoint }
