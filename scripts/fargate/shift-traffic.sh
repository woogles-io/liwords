#!/usr/bin/env bash
# Splits new requests between the EC2 and Fargate target groups.
#
#   scripts/fargate/shift-traffic.sh api 90 10      # 90% EC2, 10% Fargate
#   scripts/fargate/shift-traffic.sh socket 0 100   # all new /ws connections to Fargate
#   scripts/fargate/shift-traffic.sh api            # show the current weights
#
# api changes the listener's default action, socket the /ws rule. Neither is
# in CloudFormation (the load balancer predates it), so this changes them
# directly; rolling back is the same command with the old weights. Weights
# only steer new requests: open WebSockets and in-flight requests stay put.
set -euo pipefail

LB_NAME=woogles-prod
which="${1:?usage: shift-traffic.sh api|socket [ec2-weight fargate-weight]}"

case "$which" in
  api)    old_tg=woogles-api;       new_tg=woogles-api-fargate ;;
  socket) old_tg=woogles-websocket; new_tg=woogles-socket-fargate ;;
  *) echo "first argument must be api or socket" >&2; exit 1 ;;
esac

tg_arn() {
  aws elbv2 describe-target-groups --names "$1" --query 'TargetGroups[0].TargetGroupArn' --output text
}
old_arn=$(tg_arn "$old_tg")
new_arn=$(tg_arn "$new_tg")

lb_arn=$(aws elbv2 describe-load-balancers --names "$LB_NAME" --query 'LoadBalancers[0].LoadBalancerArn' --output text)
listener=$(aws elbv2 describe-listeners --load-balancer-arn "$lb_arn" \
  --query 'Listeners[?Port==`443`].ListenerArn | [0]' --output text)

show() {
  if [[ "$which" == api ]]; then
    aws elbv2 describe-listeners --listener-arns "$listener" \
      --query 'Listeners[0].DefaultActions[0].ForwardConfig.TargetGroups[].[TargetGroupArn,Weight]' --output text
  else
    aws elbv2 describe-rules --rule-arns "$rule" \
      --query 'Rules[0].Actions[0].ForwardConfig.TargetGroups[].[TargetGroupArn,Weight]' --output text
  fi | sed -e "s#$old_arn#$old_tg (EC2)#" -e "s#$new_arn#$new_tg (Fargate)#"
}

if [[ "$which" == socket ]]; then
  rule=$(aws elbv2 describe-rules --listener-arn "$listener" \
    --query "Rules[?Conditions[?Field=='path-pattern' && contains(Values, '/ws')]].RuleArn | [0]" --output text)
  [[ "$rule" == arn:* ]] || { echo "no /ws rule found" >&2; exit 1; }
fi

if [[ $# -eq 1 ]]; then
  show
  exit 0
fi

old_w="${2:?}"
new_w="${3:?}"
for w in "$old_w" "$new_w"; do
  [[ "$w" =~ ^[0-9]+$ && "$w" -le 999 ]] || { echo "weights are 0-999" >&2; exit 1; }
done
(( old_w + new_w > 0 )) || { echo "at least one weight must be above 0" >&2; exit 1; }

# The Fargate target group must have a healthy target before it gets any
# traffic.
if (( new_w > 0 )); then
  healthy=$(aws elbv2 describe-target-health --target-group-arn "$new_arn" \
    --query 'length(TargetHealthDescriptions[?TargetHealth.State==`healthy`])' --output text)
  (( healthy > 0 )) || { echo "$new_tg has no healthy targets; not shifting" >&2; exit 1; }
fi
if (( old_w > 0 )); then
  healthy=$(aws elbv2 describe-target-health --target-group-arn "$old_arn" \
    --query 'length(TargetHealthDescriptions[?TargetHealth.State==`healthy`])' --output text)
  (( healthy > 0 )) || { echo "$old_tg has no healthy targets; not shifting" >&2; exit 1; }
fi

actions=$(jq -nc --arg o "$old_arn" --arg n "$new_arn" --argjson ow "$old_w" --argjson nw "$new_w" \
  '[{Type: "forward", ForwardConfig: {TargetGroups: [
      {TargetGroupArn: $o, Weight: $ow}, {TargetGroupArn: $n, Weight: $nw}],
    TargetGroupStickinessConfig: {Enabled: false}}}]')

echo "Before:"; show
if [[ "$which" == api ]]; then
  aws elbv2 modify-listener --listener-arn "$listener" --default-actions "$actions" >/dev/null
else
  aws elbv2 modify-rule --rule-arn "$rule" --actions "$actions" >/dev/null
fi
echo "After:"; show
