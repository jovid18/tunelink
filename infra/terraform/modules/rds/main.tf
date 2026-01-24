locals {
  name_prefix = "${var.project_name}-${var.environment}"
}

# DB Subnet Group
resource "aws_db_subnet_group" "main" {
  name       = "${local.name_prefix}-db-subnet"
  subnet_ids = var.private_subnet_ids

  tags = {
    Name = "${local.name_prefix}-db-subnet"
  }
}

# Security Group for RDS
resource "aws_security_group" "rds" {
  name        = "${local.name_prefix}-rds-sg"
  description = "Security group for RDS"
  vpc_id      = var.vpc_id

  tags = {
    Name = "${local.name_prefix}-rds-sg"
  }
}

# Security Group Rules (분리하여 외부 모듈과 충돌 방지)
resource "aws_security_group_rule" "rds_ingress_vpc" {
  type              = "ingress"
  from_port         = 3306
  to_port           = 3306
  protocol          = "tcp"
  cidr_blocks       = ["10.0.0.0/16"]
  security_group_id = aws_security_group.rds.id
  description       = "MySQL from VPC"
}

resource "aws_security_group_rule" "rds_egress_all" {
  type              = "egress"
  from_port         = 0
  to_port           = 0
  protocol          = "-1"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.rds.id
}

# RDS Instance
resource "aws_db_instance" "main" {
  identifier = "${local.name_prefix}-mysql"

  # Engine
  engine               = "mysql"
  engine_version       = "8.0"
  instance_class       = var.instance_class

  # Storage
  allocated_storage     = var.allocated_storage
  max_allocated_storage = 100  # Auto scaling 최대값
  storage_type          = "gp2"
  storage_encrypted     = true

  # Database
  db_name  = "tunelink"
  username = var.db_username
  password = var.db_password
  port     = 3306

  # Network
  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  publicly_accessible    = false
  multi_az               = false  # dev 환경

  # Backup
  backup_retention_period = 7
  backup_window           = "03:00-04:00"  # UTC (한국시간 12:00-13:00)
  maintenance_window      = "Mon:04:00-Mon:05:00"

  # Options
  auto_minor_version_upgrade = true
  skip_final_snapshot        = true  # dev 환경 - 삭제 시 스냅샷 안 만듦
  deletion_protection        = false # dev 환경

  # Performance Insights - db.t3.micro에서 미지원
  # performance_insights_enabled = true
  # performance_insights_retention_period = 7

  tags = {
    Name = "${local.name_prefix}-mysql"
  }
}
