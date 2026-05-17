terraform {
  backend "s3" {
    bucket  = "brunojet-tfstate"
    key     = "go-edge-cache/bootstrap/terraform.tfstate"
    region  = "us-east-1"
    encrypt = true
  }
}
