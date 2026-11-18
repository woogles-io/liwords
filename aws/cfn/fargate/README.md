# liwords on Fargate

The API, socket server and bot as CloudFormation, on Fargate, in the
existing VPC and behind the existing load balancer. They run beside the EC2
services until traffic moves over, so nothing has to stop for the move.

| Stack | Template | Deployed by |
|---|---|---|
| `liwords-base` | `base.yaml` | hand, rarely |
| `liwords-api` | `liwords-api.yaml` (also the puzzle generation and migration task definitions) | CI on master (once enabled), or by hand |
| `liwords-socket` | `liwords-socket.yaml` | CI on master (once enabled), or by hand |
| `macondo-bot` | `macondo-bot.yaml` | hand (the image is built in the macondo repo) |

## What is where

- **Cluster** `woogles` (Fargate only). The EC2 cluster `woogles-prod` keeps
  running the old services until they're scaled down.
- **On-demand tasks**: `puzzle-generator-fargate`, started by the API
  (`pkg/puzzles`), and `liwords-db-migration-fargate`, which CI runs before
  each deploy. Both are defined in the `liwords-api` stack, so it has to
  exist before CI runs a migration.
- **Network**: VPC `vpc-0fe61cd12246e6ee0`, subnet `subnet-0957a537f7ee06915`
  (us-east-2a, where Postgres, Redis and NATS are). Tasks get public IPs,
  since there is no NAT gateway; the security group `liwords-fargate-tasks`
  lets in only the load balancer.
- **Load balancer**: `woogles-prod`, not in CloudFormation. The service stacks
  add their own `ip` target groups (`woogles-api-fargate`,
  `woogles-socket-fargate`) and a listener rule each that matches only a test
  header, so they get no ordinary traffic until weights are shifted.
- **Secrets**: SecureStrings under `/liwords/prod/secrets/`, read into the
  container environment when a task starts. Changing one takes effect on the
  next deploy (or `aws ecs update-service --force-new-deployment`).
- **Logs**: the same groups as the EC2 tasks (`/ecs/liwords-api`, ...), in
  streams starting `fargate/`.
- **Shell in a task**: `aws ecs execute-command --cluster woogles --task <id>
  --container liwords-api --interactive --command /bin/bash`.

## First deploy

```bash
export AWS_PROFILE=woogles-prod AWS_REGION=us-east-2
TAG=master-gh2321   # the tag the EC2 API runs now

scripts/fargate/deploy-stack.sh liwords-base   aws/cfn/fargate/base.yaml
# The task definitions, with no API task running yet:
scripts/fargate/deploy-stack.sh liwords-api    aws/cfn/fargate/liwords-api.yaml    ImageTag=$TAG DesiredCount=0
scripts/fargate/deploy-stack.sh macondo-bot    aws/cfn/fargate/macondo-bot.yaml
scripts/fargate/deploy-stack.sh liwords-socket aws/cfn/fargate/liwords-socket.yaml ImageTag=$TAG
scripts/fargate/deploy-stack.sh liwords-api    aws/cfn/fargate/liwords-api.yaml    DesiredCount=1
```

Each service starts one task. From that moment it is a second server: the
bot takes requests from the queue group, the API takes moves and runs jobs,
the socket server has no clients yet. Check them:

```bash
curl -s -H 'X-Woogles-Target: api-fargate'    https://woogles.io/ping
curl -s -H 'X-Woogles-Target: socket-fargate' https://woogles.io/ping
```

## Moving traffic

```bash
scripts/fargate/shift-traffic.sh api             # show weights
scripts/fargate/shift-traffic.sh api 90 10       # EC2 90, Fargate 10
scripts/fargate/shift-traffic.sh api 0 100
scripts/fargate/shift-traffic.sh socket 50 50
```

Roll back with the old weights. Between steps, watch 5xx and response time
on the load balancer and Jaeger. Once all traffic is on Fargate, scale the EC2
service to 0, and later delete it.

## Deploys

`.github/workflows/deploy-fargate.yml` updates `liwords-api` and
`liwords-socket` to an image tag. Stacks that don't exist are skipped.

**By hand** (the usual way: merge a batch, then deploy once). With no tag it
deploys the newest master build that pushed its images and ran its migration
(`scripts/fargate/latest-image-tag.sh`); give a tag to deploy or roll back to
that image:

```bash
gh workflow run deploy-fargate.yml                                # newest
gh workflow run deploy-fargate.yml -f image_tag=master-gh2321     # that one
```

or Actions > Deploy backend (Fargate) > Run workflow.

**On every master build**, after the migration: only if the repository
variable `FARGATE_DEPLOY` is `true`. The `production` environment can then
hold each one for approval (required reviewers); every build makes its own
waiting deploy, so approve the newest and reject the rest.

Migrations are not gated either way: every master build runs its own, so
they reach the database before their code is deployed.

**The bot** has its own workflow, `.github/workflows/deploy-bot.yml`, since
its image comes from macondo's builds. With no tag it deploys the newest
macondo master build that pushed a bot image
(`scripts/fargate/latest-bot-tag.sh`):

```bash
gh workflow run deploy-bot.yml                                    # newest
gh workflow run deploy-bot.yml -f image_tag=master-gh571          # that one
```

or Actions > Deploy macondo bot (Fargate) > Run workflow. It only runs by
hand.

CI deploys through the base stack's `liwordsCfnDeployRole`, so the CI user
can only start updates of these three stacks.

**Who can deploy:** anyone with write access to the repository can run
either workflow, unless the `production` environment has required
reviewers; then each run waits until one of them approves it.

## Retiring the EC2 instances

Once the API and socket take all traffic on Fargate, a migration has run on
Fargate, and a puzzle generation job has:

1. Scale the EC2 services (`woogles-api`, `woogles-socket`, `macondo-bot` in
   `woogles-prod`) to 0, and after a few quiet days delete them and the old
   target groups (`woogles-api`, `woogles-websocket`).
2. Set the listener's default action and the `/ws` rule to forward to the
   Fargate target groups alone.
3. Set the ECS auto scaling group to 0 and terminate the two instances
   outside it; then the cluster `woogles-prod`, `ecsInstanceRole`, and the
   old task definition families.
4. Keep the `EC2ContainerService-woogles-prod` CloudFormation stack: it owns
   the VPC, subnets, internet gateway and the security group EFS uses.
   Remove only its auto scaling group and launch configuration, if at all.

