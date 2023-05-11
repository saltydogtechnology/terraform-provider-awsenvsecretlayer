# terraform {
#   required_providers {
#     awsenvsecretlayer = {
#       source  = "terraform.local/com/awsenvsecretlayer"
#       version = "0.0.3"
#     }
#   }
# }
terraform {
  required_providers {
    awsenvsecretlayer = {
      source = "saltydogtechnology/awsenvsecretlayer"
      version = "0.0.3"
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
  layer_name = "example-layer"
  file_name  = "example.env"
  envs_map = {
    "ENV_VAR_FROM_MAP_1" = "value_1"
    "ENV_VAR_FROM_MAP_2" = "value_2"
    "ENV_VAR_FROM_MAP_3" = "value_3"
  }
  yaml_config = jsonencode(local.yaml_data)
  secrets_arns        = [
    "arn:aws:secretsmanager:us-east-1:111111111111:secret:example1/env-1/123",
    "arn:aws:secretsmanager:us-east-1:222222222222:secret:example2/secret/1233"
  ]
  compatible_runtimes = ["nodejs14.x", "python3.8"]
  skip_destroy        = true
  license_files       = ["${path.module}/envs/LICENSE.txt"]
}

output "stored_secrets" {
  value = awsenvsecretlayer_lambda.example.stored_secrets_hash
}

# re "lambda"{
#   layer_arn = awsenvsecretlayer_lambda.example.id
# }