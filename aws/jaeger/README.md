# Jaeger Production Deployment (woogles-jaeger)

Jaeger v2 with OpenSearch backend for trace storage.

## Instance Details
- **Host**: `woogles-jaeger` (configured in ~/.ssh/config)
- **Instance Type**: t3.small (2 vCPUs, 2GB RAM)
- **Disk**: 50GB gp3
- **URL**: https://jaeger.woogles.io (via Traefik)

## Architecture

```
Internet -> Traefik (port 80) -> Jaeger UI (port 16686)
                              -> Jaeger receives traces on 4317/4318
                              -> OpenSearch stores traces
```

## Files

| File | Purpose |
|------|---------|
| `docker-compose.yml` | Main compose file with all services |
| `jaeger-config-opensearch.yaml` | Jaeger v2 config for OpenSearch backend |
| `cleanup-old-indices.sh` | Cron job to delete old trace indices |
| `install-docker.sh` | One-time Docker installation script |
| `deploy.sh` | Initial deployment script |
| `monitor-disk.sh` | Disk monitoring helper |

## Deployment

### Initial Setup (new instance)

```bash
# Copy files to server
scp -r aws/jaeger/* woogles-jaeger:~/jaeger-deployment/

# SSH and deploy
ssh woogles-jaeger
cd ~/jaeger-deployment
chmod +x *.sh
./install-docker.sh
# Log out and back in, then:
./deploy.sh
```

### Update Configuration

```bash
# Copy updated files
scp aws/jaeger/docker-compose.yml woogles-jaeger:~/jaeger-deployment/
scp aws/jaeger/jaeger-config-opensearch.yaml woogles-jaeger:~/jaeger-deployment/

# Recreate containers
ssh woogles-jaeger "cd ~/jaeger-deployment && docker-compose up -d --force-recreate"
```

## Maintenance

### Index Cleanup (cron)
A cron job runs daily at 2 AM UTC to delete indices older than 1 day:
```
0 2 * * * /home/ec2-user/jaeger-deployment/cleanup-old-indices.sh >> /home/ec2-user/jaeger-deployment/cleanup.log 2>&1
```

### Disk Usage
Each day of traces uses ~8-10 GB. With 1-day retention + OpenSearch overhead:
- Expected usage: ~25-30 GB
- Safe headroom on 50GB disk

### Common Commands

```bash
# Check status
ssh woogles-jaeger "docker ps"

# View logs
ssh woogles-jaeger "docker logs jaeger-prod --tail 50"

# Check disk
ssh woogles-jaeger "df -h /"

# Check index sizes
ssh woogles-jaeger "curl -s localhost:9200/_cat/indices/jaeger-*?v"

# Restart Jaeger
ssh woogles-jaeger "cd ~/jaeger-deployment && docker-compose restart jaeger"

# Manual index cleanup (if disk full)
ssh woogles-jaeger "curl -X DELETE localhost:9200/jaeger-span-YYYY-MM-DD"
```

## Troubleshooting

### Disk Full / No Traces
If OpenSearch blocks writes due to disk:
1. Delete old indices: `curl -X DELETE localhost:9200/jaeger-span-YYYY-MM-DD`
2. Unblock indices: `curl -X PUT "localhost:9200/_all/_settings" -H "Content-Type: application/json" -d '{"index.blocks.read_only_allow_delete": null}'`

### Container Crash Loop
Check logs: `docker logs jaeger-prod`
Usually config issues - verify `jaeger-config-opensearch.yaml` is mounted correctly.

## Ports

| Port | Protocol | Service |
|------|----------|---------|
| 80 | HTTP | Traefik (routes to Jaeger UI) |
| 4317 | gRPC | OTLP traces (OpenTelemetry) |
| 4318 | HTTP | OTLP traces (OpenTelemetry) |
| 9200 | HTTP | OpenSearch (internal) |
| 14250 | gRPC | Jaeger protocol (legacy) |
| 14268 | HTTP | Jaeger Thrift (legacy) |
| 6831/udp | UDP | Jaeger Compact Thrift (legacy) |
| 6832/udp | UDP | Jaeger Binary Thrift (legacy) |

## Connecting Applications

Point your OTEL exporter to:
- **OTLP gRPC**: `jaeger.woogles.io:4317` or `<private-ip>:4317`
- **OTLP HTTP**: `http://jaeger.woogles.io:4318` or `http://<private-ip>:4318`
