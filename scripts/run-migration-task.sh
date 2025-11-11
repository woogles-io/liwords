#!/bin/bash
set -e

# Script to run the database migration ECS task and wait for completion
# Usage: ./scripts/run-migration-task.sh [OPTIONS]
#
# Options:
#   --cluster CLUSTER         ECS cluster name (default: woogles-prod)
#   --task-def TASK_DEF       Task definition family or ARN (default: liwords-db-migration)
#   --image IMAGE             Docker image to use (overrides task definition default)
#   --subnets SUBNET_IDS      Comma-separated subnet IDs (required for awsvpc mode, optional for bridge)
#   --security-groups SG_IDS  Comma-separated security group IDs (required for awsvpc mode, optional for bridge)
#   --region REGION           AWS region (default: us-east-2)
#   --launch-type TYPE        Launch type: EC2 or FARGATE (default: EC2)
#   --timeout SECONDS         Max time to wait for task completion (default: 600)
#
# Example:
#   ./scripts/run-migration-task.sh --cluster woogles-prod --task-def liwords-db-migration
#   ./scripts/run-migration-task.sh --image ghcr.io/woogles-io/liwords-api:master-gh12345

# Default values
CLUSTER="woogles-prod"
TASK_DEF="liwords-db-migration"
REGION="us-east-2"
LAUNCH_TYPE="EC2"
TIMEOUT=600
SUBNETS=""
SECURITY_GROUPS=""
IMAGE=""

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --cluster)
      CLUSTER="$2"
      shift 2
      ;;
    --task-def)
      TASK_DEF="$2"
      shift 2
      ;;
    --subnets)
      SUBNETS="$2"
      shift 2
      ;;
    --security-groups)
      SECURITY_GROUPS="$2"
      shift 2
      ;;
    --region)
      REGION="$2"
      shift 2
      ;;
    --launch-type)
      LAUNCH_TYPE="$2"
      shift 2
      ;;
    --timeout)
      TIMEOUT="$2"
      shift 2
      ;;
    --image)
      IMAGE="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [--cluster CLUSTER] [--task-def TASK_DEF] [--image IMAGE] [--subnets SUBNET_IDS] [--security-groups SG_IDS] [--region REGION] [--launch-type TYPE] [--timeout SECONDS]"
      exit 1
      ;;
  esac
done

echo "=========================================="
echo "Running Database Migration Task"
echo "=========================================="
echo "Cluster:        $CLUSTER"
echo "Task Def:       $TASK_DEF"
echo "Region:         $REGION"
echo "Launch Type:    $LAUNCH_TYPE"
echo "Timeout:        ${TIMEOUT}s"
if [[ -n "$IMAGE" ]]; then
  echo "Image Override: $IMAGE"
fi
echo "=========================================="

# Build the run-task command
RUN_TASK_CMD="aws ecs run-task \
  --cluster $CLUSTER \
  --task-definition $TASK_DEF \
  --launch-type $LAUNCH_TYPE \
  --region $REGION"

# If image override is specified, we need to register a new task definition revision
# (EC2 launch type doesn't support image overrides via --overrides parameter)
if [[ -n "$IMAGE" ]]; then
  echo "Registering new task definition with image: $IMAGE"

  # Get current task definition
  TASK_DEF_JSON=$(aws ecs describe-task-definition \
    --task-definition $TASK_DEF \
    --region $REGION \
    --query 'taskDefinition')

  # Update the image and remove read-only fields
  NEW_TASK_DEF=$(echo $TASK_DEF_JSON | jq \
    --arg IMAGE "$IMAGE" \
    '.containerDefinitions[0].image = $IMAGE | del(.taskDefinitionArn, .revision, .status, .requiresAttributes, .compatibilities, .registeredAt, .registeredBy)')

  # Register new task definition revision
  NEW_TASK_DEF_ARN=$(aws ecs register-task-definition \
    --cli-input-json "$NEW_TASK_DEF" \
    --region $REGION \
    --query 'taskDefinition.taskDefinitionArn' \
    --output text)

  echo "Created new task definition revision: $NEW_TASK_DEF_ARN"

  # Use the new task definition ARN instead
  TASK_DEF="$NEW_TASK_DEF_ARN"
  RUN_TASK_CMD="aws ecs run-task --cluster $CLUSTER --task-definition $TASK_DEF --launch-type $LAUNCH_TYPE --region $REGION"
fi

# Add network configuration if subnets/security groups are provided
if [[ -n "$SUBNETS" ]] || [[ -n "$SECURITY_GROUPS" ]]; then
  NETWORK_CONFIG="{"
  if [[ -n "$SUBNETS" ]]; then
    NETWORK_CONFIG="${NETWORK_CONFIG}\"awsvpcConfiguration\":{\"subnets\":[$(echo $SUBNETS | sed 's/,/","/g' | sed 's/^/"/;s/$/"/')],\"assignPublicIp\":\"DISABLED\""
    if [[ -n "$SECURITY_GROUPS" ]]; then
      NETWORK_CONFIG="${NETWORK_CONFIG},\"securityGroups\":[$(echo $SECURITY_GROUPS | sed 's/,/","/g' | sed 's/^/"/;s/$/"/')]}}"
    else
      NETWORK_CONFIG="${NETWORK_CONFIG}}}"
    fi
  fi
  RUN_TASK_CMD="$RUN_TASK_CMD --network-configuration '$NETWORK_CONFIG'"
fi

# Run the task
echo "Starting migration task..."
TASK_OUTPUT=$(eval $RUN_TASK_CMD)

# Extract task ARN
TASK_ARN=$(echo $TASK_OUTPUT | jq -r '.tasks[0].taskArn')

if [[ -z "$TASK_ARN" ]] || [[ "$TASK_ARN" == "null" ]]; then
  echo "ERROR: Failed to start task"
  echo "$TASK_OUTPUT" | jq .
  exit 1
fi

TASK_ID=$(echo $TASK_ARN | awk -F/ '{print $NF}')
echo "Task started: $TASK_ID"
echo "Task ARN: $TASK_ARN"

# Wait for task to complete
echo ""
echo "Waiting for task to complete (timeout: ${TIMEOUT}s)..."

ELAPSED=0
POLL_INTERVAL=10

while [ $ELAPSED -lt $TIMEOUT ]; do
  # Get task status
  TASK_STATUS=$(aws ecs describe-tasks \
    --cluster $CLUSTER \
    --tasks $TASK_ARN \
    --region $REGION \
    --query 'tasks[0].lastStatus' \
    --output text)

  echo "[${ELAPSED}s] Task status: $TASK_STATUS"

  # Check if task has stopped
  if [ "$TASK_STATUS" == "STOPPED" ]; then
    echo ""
    echo "Task completed. Checking exit code..."

    # Get container exit code
    EXIT_CODE=$(aws ecs describe-tasks \
      --cluster $CLUSTER \
      --tasks $TASK_ARN \
      --region $REGION \
      --query 'tasks[0].containers[0].exitCode' \
      --output text)

    # Get stop reason
    STOP_REASON=$(aws ecs describe-tasks \
      --cluster $CLUSTER \
      --tasks $TASK_ARN \
      --region $REGION \
      --query 'tasks[0].stoppedReason' \
      --output text)

    echo "Exit code: $EXIT_CODE"
    echo "Stop reason: $STOP_REASON"

    if [ "$EXIT_CODE" == "0" ]; then
      echo ""
      echo "=========================================="
      echo "✓ Migration completed successfully!"
      echo "=========================================="
      exit 0
    else
      echo ""
      echo "=========================================="
      echo "✗ Migration failed with exit code $EXIT_CODE"
      echo "=========================================="

      # Try to fetch recent logs
      echo ""
      echo "Fetching recent CloudWatch logs..."
      LOG_GROUP="/ecs/liwords-db-migration"
      LOG_STREAM=$(aws logs describe-log-streams \
        --log-group-name $LOG_GROUP \
        --region $REGION \
        --order-by LastEventTime \
        --descending \
        --max-items 1 \
        --query 'logStreams[0].logStreamName' \
        --output text 2>/dev/null || echo "")

      if [[ -n "$LOG_STREAM" ]] && [[ "$LOG_STREAM" != "None" ]]; then
        echo "Log stream: $LOG_STREAM"
        echo "---"
        aws logs get-log-events \
          --log-group-name $LOG_GROUP \
          --log-stream-name $LOG_STREAM \
          --region $REGION \
          --limit 50 \
          --query 'events[*].message' \
          --output text 2>/dev/null || echo "Could not fetch logs"
      else
        echo "No log stream found. Check CloudWatch Logs manually:"
        echo "https://console.aws.amazon.com/cloudwatch/home?region=$REGION#logsV2:log-groups/log-group/$LOG_GROUP"
      fi

      exit 1
    fi
  fi

  # Wait before polling again
  sleep $POLL_INTERVAL
  ELAPSED=$((ELAPSED + POLL_INTERVAL))
done

# Timeout reached
echo ""
echo "=========================================="
echo "✗ Timeout reached after ${TIMEOUT}s"
echo "=========================================="
echo "Task may still be running. Check the AWS Console or run:"
echo "  aws ecs describe-tasks --cluster $CLUSTER --tasks $TASK_ARN --region $REGION"
exit 1
