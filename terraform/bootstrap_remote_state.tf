// Read outputs from the bootstrap stack (bootstrap/terraform.tfstate)
data "terraform_remote_state" "bootstrap" {
  backend = "s3"
  config = {
    bucket = "brunojet-tfstate"
    key    = "bootstrap/terraform.tfstate"
    region = var.aws_region
  }
}

locals {
  bootstrap_secret_arn  = try(data.terraform_remote_state.bootstrap.outputs.secret_arn, "")
  bootstrap_secret_id   = try(data.terraform_remote_state.bootstrap.outputs.secret_id, "")
  bootstrap_iam_role_arn = try(data.terraform_remote_state.bootstrap.outputs.iam_role_arn, "")
  bootstrap_s3_bucket   = try(data.terraform_remote_state.bootstrap.outputs.s3_bucket_name, "")
}
