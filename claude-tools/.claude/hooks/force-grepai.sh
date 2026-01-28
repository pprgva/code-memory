#!/bin/bash
# =============================================================================
# Hook: force-grepai.sh
# Purpose: Force Claude to use grepai instead of grep/glob/find for searches
# =============================================================================

set -euo pipefail

INPUT=$(cat)
TOOL=$(echo "$INPUT" | jq -r '.tool_name // ""')
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // ""')

# -----------------------------------------------------------------------------
# Configuration: patterns to intercept
# -----------------------------------------------------------------------------

# Bash commands that should be replaced by grepai
SEARCH_COMMANDS="grep|egrep|fgrep|rg|ripgrep|ag|ack|pt|sift"
FIND_COMMANDS="find.*-name|find.*-iname|locate|fd"

# Combined pattern
BLOCKED_PATTERNS="($SEARCH_COMMANDS|$FIND_COMMANDS)"

# -----------------------------------------------------------------------------
# Logic
# -----------------------------------------------------------------------------

# Check if grepai is available in this project
has_grepai_config() {
    [[ -f ".grepai/config.yaml" ]] || [[ -f ".grepai/config.yml" ]]
}

# Block glob tool → use grepai search instead
if [[ "$TOOL" == "glob" ]]; then
    if has_grepai_config; then
        cat <<EOF
{
  "decision": "block",
  "reason": "🚫 STOP - N'utilise PAS glob. Utilise grepai search à la place.\n\nExemple: grepai search 'ton intention en anglais' --json --compact\n\nSi tu cherches des fichiers par pattern, décris ce que tu cherches en langage naturel."
}
EOF
        exit 0
    fi
fi

# Block grep/find/rg commands in bash → use grepai search instead
if [[ "$TOOL" == "bash" ]]; then
    if echo "$COMMAND" | grep -qE "$BLOCKED_PATTERNS"; then
        if has_grepai_config; then
            # Extract what seems to be the search term for helpful suggestion
            SEARCH_TERM=$(echo "$COMMAND" | grep -oE '"[^"]+"' | head -1 || echo "<ta requête>")
            
            cat <<EOF
{
  "decision": "block",
  "reason": "🚫 STOP - N'utilise PAS grep/find/rg. Ce projet utilise grepai.\n\nRemplace par:\n  grepai search 'description en anglais' --json --compact\n\nExemple pour ta recherche:\n  grepai search ${SEARCH_TERM} --json --compact\n\nPour tracer les appels:\n  grepai trace callers 'Symbol' --json\n  grepai trace callees 'Symbol' --json"
}
EOF
            exit 0
        fi
    fi
fi

# Allow everything else
echo '{"decision": "approve"}'
