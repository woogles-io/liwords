#!/usr/bin/env bash
# Creates or updates one of the Fargate stacks in aws/cfn/fargate.
#
#   scripts/fargate/deploy-stack.sh liwords-base   aws/cfn/fargate/base.yaml
#   scripts/fargate/deploy-stack.sh liwords-api    aws/cfn/fargate/liwords-api.yaml ImageTag=master-gh2321
#   scripts/fargate/deploy-stack.sh liwords-socket aws/cfn/fargate/liwords-socket.yaml ImageTag=master-gh2321
#   scripts/fargate/deploy-stack.sh macondo-bot    aws/cfn/fargate/macondo-bot.yaml
#
# Parameters not given keep their current value on an existing stack (their
# default on a new one). Service stacks are deployed through the base stack's
# deploy role, the same way CI deploys them. An unchanged stack is a no-op.
#
# Set REQUIRE_EXISTING=1 to skip, rather than create, a stack that doesn't
# exist yet (CI does, so merging a template doesn't create a service).
set -euo pipefail

STACK="${1:?usage: deploy-stack.sh <stack> <template> [Key=Value ...]}"
TEMPLATE="${2:?usage: deploy-stack.sh <stack> <template> [Key=Value ...]}"
shift 2
BASE_STACK="${BASE_STACK:-liwords-base}"

status=$(aws cloudformation describe-stacks --stack-name "$STACK" \
  --query 'Stacks[0].StackStatus' --output text 2>/dev/null || echo NONE)

if [[ "$status" == NONE && "${REQUIRE_EXISTING:-0}" == 1 ]]; then
  echo "::notice::stack $STACK doesn't exist yet; skipping"
  exit 0
fi
if [[ "$status" == *_IN_PROGRESS ]]; then
  echo "stack $STACK is busy ($status); try again when it finishes" >&2
  exit 1
fi

args=(--stack-name "$STACK" --template-file "$TEMPLATE"
  --capabilities CAPABILITY_NAMED_IAM --no-fail-on-empty-changeset
  --tags app=liwords)
if [[ $# -gt 0 ]]; then
  args+=(--parameter-overrides "$@")
fi
if [[ "$STACK" != "$BASE_STACK" ]]; then
  role=$(aws cloudformation describe-stacks --stack-name "$BASE_STACK" \
    --query "Stacks[0].Outputs[?OutputKey=='DeployRoleArn'].OutputValue" --output text)
  args+=(--role-arn "$role")
fi

echo "Deploying $STACK ($status) from $TEMPLATE $*"
if ! aws cloudformation deploy "${args[@]}"; then
  echo "Deploy of $STACK failed. Recent failures:" >&2
  aws cloudformation describe-stack-events --stack-name "$STACK" --max-items 30 \
    --query 'StackEvents[?contains(ResourceStatus, `FAILED`)].[Timestamp,LogicalResourceId,ResourceStatusReason]' \
    --output text >&2 || true
  exit 1
fi
aws cloudformation describe-stacks --stack-name "$STACK" \
  --query 'Stacks[0].[StackStatus, Outputs]' --output json
