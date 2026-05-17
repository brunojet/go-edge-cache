provider "aws" {
  region = var.region
}

module "secrets" {
  source        = "../modules/secrets"
  create        = var.create_secrets
  name          = var.secret_name
  secret_string = var.secret_string
  tags          = {}
}


