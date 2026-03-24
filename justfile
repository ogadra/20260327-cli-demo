# Terraform operations with environment-specific tfvars

# Plan changes for the specified environment
tf-plan env:
    cd terraform && terraform plan -var-file=environments/{{env}}.tfvars

# Apply changes for the specified environment
tf-apply env:
    cd terraform && terraform apply -var-file=environments/{{env}}.tfvars

# Initialize terraform
tf-init:
    cd terraform && terraform init

# Destroy resources for the specified environment
tf-destroy env:
    cd terraform && terraform destroy -var-file=environments/{{env}}.tfvars
