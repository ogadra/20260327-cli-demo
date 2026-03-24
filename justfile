# Terraform operations with environment-specific tfvars
# Available environments: stg, prd

# Initialize terraform
init:
    terraform -chdir=terraform init

# Plan changes for the specified environment
plan env:
    terraform -chdir=terraform plan -var-file=environments/{{env}}.tfvars

# Apply changes for the specified environment
apply env:
    terraform -chdir=terraform apply -var-file=environments/{{env}}.tfvars

# Destroy resources for the specified environment
destroy env:
    terraform -chdir=terraform destroy -var-file=environments/{{env}}.tfvars
