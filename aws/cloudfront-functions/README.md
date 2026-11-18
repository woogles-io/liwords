# CloudFront Functions

`page-gate.js` is the viewer-request function `woogles-page-gate`. It runs on
the production distribution's `/game/*`, `/profile/*` and `/anno/*` cache
behaviors. Those behaviors and the function association were set up by hand
and are not in `cfn/cloudfront.yaml`, so redeploying that template would
drop them.

The function checks the `wg_ok` cookie that `/api/human-check`
(`pkg/humancheck`) sets. Both sides need the same key: the API reads it from
`HUMAN_CHECK_COOKIE_KEY`, and it is substituted for
`__HUMAN_CHECK_COOKIE_KEY__` when the function is uploaded. Never commit the
real key.

## Deploying

`update-function` only changes the DEVELOPMENT stage; nothing reaches viewers
until `publish-function`. Set `TASK_DEF` to the API's current task
definition.

```bash
export AWS_PROFILE=woogles-prod
KEY=$(aws ecs describe-task-definition --task-definition "$TASK_DEF" \
  --query "taskDefinition.containerDefinitions[0].environment[?name=='HUMAN_CHECK_COOKIE_KEY'].value" \
  --output text)
ETAG=$(aws cloudfront describe-function --name woogles-page-gate --query ETag --output text)
sed "s|__HUMAN_CHECK_COOKIE_KEY__|$KEY|" page-gate.js | aws cloudfront update-function \
  --name woogles-page-gate --if-match "$ETAG" \
  --function-config '{"Comment":"Edge checks for page loads","Runtime":"cloudfront-js-2.0"}' \
  --function-code fileb:///dev/stdin
unset KEY

# Try it on DEVELOPMENT with test events, e.g.:
#   aws cloudfront test-function --name woogles-page-gate --stage DEVELOPMENT \
#     --if-match <etag> --event-object fileb://event.json

aws cloudfront publish-function --name woogles-page-gate \
  --if-match "$(aws cloudfront describe-function --name woogles-page-gate --query ETag --output text)"
```

To check what is live, `aws cloudfront get-function --name woogles-page-gate
--stage LIVE out.js` downloads it, key included.
