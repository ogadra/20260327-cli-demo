# Terraform operations with environment-specific tfvars
# Available environments: stg, prd

_validate-env env:
    @if [ "{{env}}" != "stg" ] && [ "{{env}}" != "prd" ]; then echo "Error: env must be 'stg' or 'prd', got '{{env}}'"; exit 1; fi

# Initialize terraform
init:
    terraform -chdir=terraform init

# Plan changes for the specified environment
plan env: (_validate-env env)
    terraform -chdir=terraform plan -var-file=environments/{{env}}.tfvars

# Apply changes for the specified environment
apply env: (_validate-env env)
    terraform -chdir=terraform apply -var-file=environments/{{env}}.tfvars

# Destroy resources for the specified environment
destroy env: (_validate-env env)
    terraform -chdir=terraform destroy -var-file=environments/{{env}}.tfvars
