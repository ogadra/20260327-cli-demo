"""Custom Checkov policy to enforce Project=Bunshin tag on all taggable resources."""

from typing import Any

from checkov.terraform.checks.resource.base_resource_check import BaseResourceCheck
from checkov.common.models.enums import CheckCategories, CheckResult


class BunshinTagPolicy(BaseResourceCheck):
    """Ensure all taggable resources have the Project=Bunshin tag for cost management."""

    def __init__(self) -> None:
        """Initialize the check with ID, supported resources, and category."""
        name = "Ensure Project=Bunshin tag is present"
        check_id = "CKV_BUNSHIN_1"
        supported_resources = ["*"]
        categories = [CheckCategories.CONVENTION]
        super().__init__(name=name, id=check_id, categories=categories, supported_resources=supported_resources)

    def scan_resource_conf(self, conf: dict[str, list[Any]]) -> CheckResult:
        """Check that the resource has a Project=Bunshin tag."""
        tags = conf.get("tags", [{}])
        if isinstance(tags, list):
            tags = tags[0] if tags else {}
        if not isinstance(tags, dict):
            return CheckResult.UNKNOWN
        project_value = tags.get("Project", "")
        if isinstance(project_value, list) and project_value:
            project_value = project_value[0]
        if project_value == "Bunshin":
            return CheckResult.PASSED
        return CheckResult.FAILED


check = BunshinTagPolicy()
