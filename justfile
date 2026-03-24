# Terraform operations with environment-specific tfvars

# Plan changes for the specified environment
tf-plan env:
    terraform -chdir=terraform plan -var-file=environments/{{env}}.tfvars

# Apply changes for the specified environment
tf-apply env:
    terraform -chdir=terraform apply -var-file=environments/{{env}}.tfvars

# Initialize terraform
tf-init:
    terraform -chdir=terraform init

# Destroy resources for the specified environment
tf-destroy env:
    terraform -chdir=terraform destroy -var-file=environments/{{env}}.tfvars
