locals {
  repositories = ["api", "web"]
}

resource "aws_ecr_repository" "main" {
  for_each = toset(local.repositories)

  name                 = "${var.project_name}-${each.key}"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = {
    Name = "${var.project_name}-${each.key}"
  }
}

# 오래된 이미지 자동 삭제 (비용 절감)
resource "aws_ecr_lifecycle_policy" "main" {
  for_each = toset(local.repositories)

  repository = aws_ecr_repository.main[each.key].name

  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Keep last 10 images"
        selection = {
          tagStatus   = "any"
          countType   = "imageCountMoreThan"
          countNumber = 10
        }
        action = {
          type = "expire"
        }
      }
    ]
  })
}
