#!/usr/bin/env bash
# Applies the database migrations in an API image, by running that image's
# migrations once as a Fargate task, and waits for it.
#
#   scripts/fargate/run-migration.sh master-gh2342
#
# It registers a revision of liwords-db-migration-fargate (the liwords-api
# stack) with the image, and runs it in the cluster and network of
# liwords-base. Migrations already applied are skipped, so running it twice,
# or for an older image, does nothing.
set -euo pipefail

tag="${1:?usage: run-migration.sh <image tag>}"
TASK_DEFINITION=liwords-db-migration-fargate
LOG_GROUP=/ecs/liwords-db-migration
image="ghcr.io/woogles-io/liwords-api:$tag"

base_output() {
  aws cloudformation describe-stacks --stack-name liwords-base \
    --query "Stacks[0].Outputs[?OutputKey=='$1'].OutputValue" --output text
}
cluster=$(base_output ClusterName)
network="awsvpcConfiguration={subnets=[$(base_output Subnets)],securityGroups=[$(base_output TaskSecurityGroupId)],assignPublicIp=ENABLED}"

def=$(aws ecs describe-task-definition --task-definition "$TASK_DEFINITION" --query taskDefinition |
  jq --arg image "$image" '.containerDefinitions[0].image = $image
    | del(.taskDefinitionArn, .revision, .status, .requiresAttributes, .compatibilities, .registeredAt, .registeredBy)')
def_arn=$(aws ecs register-task-definition --cli-input-json "$def" \
  --query taskDefinition.taskDefinitionArn --output text)
echo "Running $def_arn ($image) on $cluster"

task=$(aws ecs run-task --cluster "$cluster" --task-definition "$def_arn" --launch-type FARGATE \
  --network-configuration "$network" --query 'tasks[0].taskArn' --output text)
if [[ -z "$task" || "$task" == None ]]; then
  echo "the migration task didn't start" >&2
  exit 1
fi
echo "Migration task ${task##*/} started"

# Migrations take seconds; ten minutes means something is stuck.
if ! aws ecs wait tasks-stopped --cluster "$cluster" --tasks "$task"; then
  echo "the migration task hasn't stopped after the waiter's timeout" >&2
  exit 1
fi
read -r code reason < <(aws ecs describe-tasks --cluster "$cluster" --tasks "$task" \
  --query 'tasks[0].[containers[0].exitCode, stoppedReason]' --output text)
echo "Migration task exited with $code ($reason)"
if [[ "$code" == 0 ]]; then
  exit 0
fi

echo "Its log:" >&2
aws logs get-log-events --log-group-name "$LOG_GROUP" \
  --log-stream-name "fargate/db-migration/${task##*/}" --limit 100 \
  --query 'events[].message' --output text >&2 || echo "(couldn't read the log)" >&2
exit 1
