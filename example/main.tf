terraform {
  required_providers {
    awsenvsecretlayer = {
      source = "saltydogtechnology/awsenvsecretlayer"
      version = "0.0.1"
    }
  }
}

provider "awsenvsecretlayer" {
  region  = "eu-central-1"
  profile = "sso-dg-sandbox"
}

# output "yamldecode1" {
#   value = yamldecode(file("${path.module}/envs/merged.yaml"))
# }

locals {
  yaml_data = yamldecode(file("${path.module}/envs/merged.yaml"))
}

resource "awsenvsecretlayer_lambda" "example" {
  layer_name  = "example-layer"
  file_name   = "example.env"
  yaml_config = jsonencode(local.yaml_data)
  secrets_arns = [
    "arn:aws:secretsmanager:eu-central-1:582972330974:secret:dg/sandbox/plex-2Z10kb",
    "arn:aws:secretsmanager:eu-central-1:582972330974:secret:example-secret-2mYP9T"
  ]
  compatible_runtimes = ["nodejs14.x", "python3.8"]
  skip_destroy        = false
  license_files       = ["${path.module}/envs/LICENSE.txt"]
}

output "stored_secrets" {
  value = awsenvsecretlayer_lambda.example.stored_secrets_hash
}

# re "lambda"{
#   layer_arn = awsenvsecretlayer_lambda.example.id
# }