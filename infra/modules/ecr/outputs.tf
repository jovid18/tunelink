output "repository_urls" {
  description = "ECR repository URLs"
  value = {
    for key, repo in aws_ecr_repository.main : key => repo.repository_url
  }
}

output "repository_arns" {
  description = "ECR repository ARNs"
  value = {
    for key, repo in aws_ecr_repository.main : key => repo.arn
  }
}
