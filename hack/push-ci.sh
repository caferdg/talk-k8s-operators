#!/usr/bin/env bash
set -euo pipefail

# Usage: push-ci.sh <cr-name> <pass|fail>

CR="${1:?Usage: push-ci.sh <cr-name> <pass|fail>}"
MODE="${2:?Usage: push-ci.sh <cr-name> <pass|fail>}"
GITLAB_API="https://gitlab.com/api/v4"

TOKEN=$(kubectl get secret gitlab-token -o jsonpath='{.data.token}' | base64 -d)
FULL_PATH=$(kubectl get gitlabproject "$CR" -o jsonpath='{.status.fullPath}')
PROJECT_ID=$(curl -s -H "PRIVATE-TOKEN: $TOKEN" "$GITLAB_API/projects/$(echo "$FULL_PATH" | sed 's|/|%2F|g')" | jq '.id')
BRANCH=$(kubectl get gitlabproject "$CR" -o jsonpath='{.spec.trackedBranch}')

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

if [ "$MODE" = "pass" ]; then
  SCRIPT='sleep 15 && echo \"All good!\"'
  LABEL="passing"
  COLOR_LABEL="${GREEN}passing${NC}"
else
  SCRIPT='sleep 15 && exit 1'
  LABEL="failing"
  COLOR_LABEL="${RED}failing${NC}"
fi

CI_CONTENT="default:\\n  image: alpine:latest\\n  tags:\\n    - saas-linux-small-amd64\\n\\ntest:\\n  script:\\n    - $SCRIPT"

echo -e "Pushing $COLOR_LABEL .gitlab-ci.yml to $FULL_PATH ($BRANCH)..."

# Try update first, fall back to create
RESULT=$(curl -s --request POST -H "PRIVATE-TOKEN: $TOKEN" -H "Content-Type: application/json" \
  "$GITLAB_API/projects/$PROJECT_ID/repository/commits" \
  -d "{\"branch\": \"$BRANCH\", \"commit_message\": \"ci: $LABEL pipeline\", \"actions\": [{\"action\": \"update\", \"file_path\": \".gitlab-ci.yml\", \"content\": \"$CI_CONTENT\"}]}")

if echo "$RESULT" | jq -e '.id' > /dev/null 2>&1; then
  echo "$RESULT" | jq '{id: .id, message: .message, web_url: .web_url}'
else
  curl -s --request POST -H "PRIVATE-TOKEN: $TOKEN" -H "Content-Type: application/json" \
    "$GITLAB_API/projects/$PROJECT_ID/repository/commits" \
    -d "{\"branch\": \"$BRANCH\", \"commit_message\": \"ci: $LABEL pipeline\", \"actions\": [{\"action\": \"create\", \"file_path\": \".gitlab-ci.yml\", \"content\": \"$CI_CONTENT\"}]}" \
    | jq '{id: .id, message: .message, web_url: .web_url}'
fi
