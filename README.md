
# AWS Lambda Environment Secret Layer Terraform Provider

This Terraform provider offers a custom resource for managing AWS Lambda environment secret layers. It allows you to create and update Lambda layers with environment variables and secrets from AWS Secrets Manager. The layer is created with a **.env** file containing the environment variables and secrets, which can then be used by your Lambda functions.

## Features
- Creates a Lambda layer with environment variables and secrets.
- Supports updating the Lambda layer when changes are detected in environment variables or secrets.
- Allows controlling the deletion of the Lambda layer during the update process with the **skip_destroy** parameter.

## Usage

##### **`./envs/vars.yaml`**
```
var1: "example-1"
var2: "example-2"
var3: "example-3"
```

##### **`main.tf`**
```
terraform {
  required_providers {
    awsenvsecretlayer = {
      source = "saltydogtechnology/awsenvsecretlayer"
      version = "1.0.1"
    }
  }
}

provider "awsenvsecretlayer" {
  region  = "us-east-1"
  profile = "aws-profile-name"
}

locals {
  yaml_data = yamldecode(file("${path.module}/envs/vars.yaml"))
}

resource "awsenvsecretlayer_lambda" "example" {
  layer_name          = "example-layer"
  file_name           = "example.env"
  yaml_config         = jsonencode(local.yaml_data)
  secrets_arns        = [
    "arn:aws:secretsmanager:us-east-1:111111111111:secret:example1/env-1/123",
    "arn:aws:secretsmanager:us-east-1:222222222222:secret:example2/secret/1233"
  ]
  envs_map = {
    "ENV_VAR_FROM_MAP_1" = "value_1"
    "ENV_VAR_FROM_MAP_2" = "value_2"
    "ENV_VAR_FROM_MAP_3" = "value_3"
  }
  compatible_runtimes = ["nodejs14.x", "python3.8"]
  skip_destroy        = false
  license_files       = ["${path.module}/envs/LICENSE.txt"]
}
```

## Inputs
<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Description</th>
      <th>Type</th>
      <th>Default</th>
      <th>Required</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>layer_name</td>
      <td>Name of the Lambda Layer.</td>
      <td>string</td>
      <td>n/a</td>
      <td>yes</td>
    </tr>
    <tr>
      <td>file_name</td>
      <td>Name of the environment file within the Lambda Layer.</td>
      <td>string</td>
      <td>n/a</td>
      <td>yes</td>
    </tr>
    <tr>
      <td>yaml_config</td>
      <td>YAML configuration content, as a string.</td>
      <td>string</td>
      <td>""</td>
      <td>no</td>
    </tr>
    <tr>
      <td>secrets_arns</td>
      <td>List of AWS Secrets Manager ARNs to fetch secrets from.</td>
      <td>list(string)</td>
      <td>[]</td>
      <td>no</td>
    </tr>
    <tr>
      <td>envs_map</td>
      <td>A map of environment variables to be included in the AWS Lambda Layer .env file. </td>
      <td>map(string)</td>
      <td>{}</td>
      <td>no</td>
    </tr>
    <tr>
      <td>compatible_runtimes</td>
      <td>List of compatible runtimes for the Lambda Layer.</td>
      <td>list(string)</td>
      <td>[]</td>
      <td>no</td>
    </tr>
    <tr>
      <td>skip_destroy</td>
      <td>Whether to skip deleting the layer version during updates.</td>
      <td>bool</td>
      <td>false</td>
      <td>no</td>
    </tr>
    <tr>
      <td>license_files</td>
      <td>A list of file paths for license files that you want to include in the layer.</td>
      <td>list(string)</td>
      <td>[]</td>
      <td>no</td>
    </tr>
  </tbody>
</table>

## Outputs
<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Description</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>layer_id</td>
      <td>The ARN of the created Lambda layer.</td>
    </tr>
  </tbody>
</table>

## Limitations
- The module does not support reading the existing Lambda layer, as the API does not provide information that can be used for this purpose.
- The plan output does not show "1 to destroy" when a layer is deleted during an update, as Terraform considers it an update rather than a delete/create operation.
