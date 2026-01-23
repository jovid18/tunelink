terraform {
  backend "s3" {
    bucket         = "tunelink-terraform-state-058264445568"
    key            = "dev/terraform.tfstate"
    region         = "ap-northeast-2"
    dynamodb_table = "tunelink-terraform-lock"
    encrypt        = true
  }
}
