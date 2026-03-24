# Terraform operations with environment-specific tfvars

# Plan changes for the specified environment
plan env:
    terraform -chdir=terraform plan -var-file=environments/{{env}}.tfvars

# Apply changes for the specified environment
apply env:
    terraform -chdir=terraform apply -var-file=environments/{{env}}.tfvars

# Initialize terraform
init:
    terraform -chdir=terraform init

# Destroy resources for the specified environment
destroy env:
    terraform -chdir=terraform destroy -var-file=environments/{{env}}.tfvars
