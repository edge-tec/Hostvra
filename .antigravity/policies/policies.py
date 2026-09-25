"""
Hostvra AI Engineering — Policy Configuration Files
Defines the three policy tiers as YAML-like Python dicts.
"""

# READ-ONLY POLICY
# Use for: repository analysis, architecture review, code reading
# Allows: view_file, list_directory, search_directory, find_file
# Blocks: run_command, create_file, edit_file
READ_ONLY = {
    "name": "read_only",
    "description": "Read-only analysis — no shell execution, no file writes",
    "allowed_tools": ["view_file", "list_directory", "search_directory", "find_file", "finish"],
    "blocked_tools": ["run_command", "create_file", "edit_file"],
    "requires_approval": [],
}

# ANALYSIS POLICY
# Use for: security audits, diagnostics, test running, code quality checks
# Allows: all read tools + SAFE shell commands (classified by command_safety.py)
# Blocks: DANGEROUS/BLOCKED commands, file writes
ANALYSIS = {
    "name": "analysis",
    "description": "Analysis — safe read-only commands + file reading, no writes",
    "allowed_tools": ["view_file", "list_directory", "search_directory", "find_file", "finish"],
    "blocked_tools": ["create_file", "edit_file"],
    "requires_approval_tools": ["run_command"],
    "command_policy": "SAFE_ONLY",  # Only SAFE-classified commands pass
}

# DEVELOPMENT POLICY
# Use for: code modifications, adding features, fixing bugs
# Allows: all tools with approval gates for destructive operations
# DANGEROUS/BLOCKED commands are still blocked
DEVELOPMENT = {
    "name": "development",
    "description": "Development — file writes allowed with approval, DANGEROUS commands blocked",
    "allowed_tools": ["view_file", "list_directory", "search_directory", "find_file", "finish"],
    "requires_approval_tools": ["run_command", "create_file", "edit_file"],
    "command_policy": "REVIEW_WITH_APPROVAL",  # SAFE=auto, REVIEW=ask, DANGEROUS=deny
    "post_change_requirements": [
        "go vet ./... (for Go changes)",
        "go test -count=1 ./... (for Go changes)",
        "npm run build (for TypeScript changes)",
    ],
}
