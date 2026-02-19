---
provider: brave
mode: web
mode_adjusted: true
truncated: false
sources_count: 5
cached: false
---

**Using AWS AssumeRole with the AWS Terraform Provider – HashiCorp Help Center**
In your Terraform configuration, **configure the AWS provider to use the credentials for Account A and specify the assume_role block to connect to Account B**. provider "aws" { ## Credentials for the IAM user in AWS Account A. ## We recommend using ...

**Authenticate providers with dynamic credentials | Terraform | HashiCorp Developer**
Then the configuration defines an IAM role for the trust relationship. **If the request meets the role's conditions, AWS will provide HCP Terraform with dynamic credentials that assume this role**.

**amazon web services - Terraform Cloud / Enterprise - How to use AWS Assume Roles - Stack Overflow**
See the announcement and the official docs: Dynamic Credentials with the AWS Provider ... I used the exact same provider configuration minus the explicit adding of the acces keys. The access keys were added in the Terraform Cloud workspace as environment variables. ... This is definitely possible with Terraform Enterprise (TFE) if your TFE infrastructure is also hosted in AWS and the instance profile is trusted by the role you are trying to assume.

**AWS Provider - Terraform Registry**
The AWS Provider supports assuming an IAM role using web identity federation and OpenID Connect (OIDC). **This can be configured either using environment variables or in a named profile**. When using a named profile, the AWS Provider also supports sourcing credentials from an external process.

**Cross Account Terraform Assume Roles in AWS | Ruan Bekker's Blog**
In this tutorial we will go through a scenario where we use Terraform to write our remote state to a s3 bucket in our Tools Account, our main user is configured in our Management Account and we will deploy a SNS Topic in our Dev Account. We will accomplish this with assume roles. ... provider "aws" { region = "eu-west-1" profile = "management" shared_credentials_files = ["~/.aws/credentials"] } provider "aws" { alias = "dev" region = "eu-west-1" profile = "management" shared_credentials_files = ["~/.aws/credentials"] assume_role { role_arn = "arn:aws:iam::000000000003:role/terraform-role" } } terraform { backend "s3" { encrypt = true bucket = "terraform-state-tools-account" key = "example/terraform.tfstate" region = "eu-west-1" profile = "management" shared_credentials_files = ["~/.aws/credentials"] assume_role = { role_arn = "arn:aws:iam::000000000002:role/terraform-role" } } }