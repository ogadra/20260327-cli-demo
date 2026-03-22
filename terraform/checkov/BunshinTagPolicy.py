"""Custom Checkov policy to enforce Project=Bunshin tag on all taggable resources."""

from checkov.terraform.checks.resource.base_resource_check import BaseResourceCheck
from checkov.common.models.enums import CheckCategories, CheckResult


class BunshinTagPolicy(BaseResourceCheck):
    """Ensure all taggable resources have the Project=Bunshin tag for cost management."""

    def __init__(self) -> None:
        name = "Ensure Project=Bunshin tag is present"
        id = "CKV_BUNSHIN_1"
        supported_resources = ["*"]
        categories = [CheckCategories.GENERAL_SECURITY]
        super().__init__(name=name, id=id, categories=categories, supported_resources=supported_resources)

    def scan_resource_conf(self, conf: dict) -> CheckResult:
        """Check that the resource has a Project=Bunshin tag."""
        tags = conf.get("tags", [{}])
        if isinstance(tags, list):
            tags = tags[0] if tags else {}
        if not isinstance(tags, dict):
            return CheckResult.UNKNOWN
        if tags.get("Project") == ["Bunshin"] or tags.get("Project") == "Bunshin":
            return CheckResult.PASSED
        return CheckResult.FAILED


check = BunshinTagPolicy()
