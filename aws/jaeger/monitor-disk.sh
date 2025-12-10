#!/bin/bash

echo "=== Jaeger Disk Usage Monitor ==="
echo "Date: $(date)"
echo ""

echo "=== Overall Disk Space ==="
df -h / | tail -1
echo ""

echo "=== Badger Data Size ==="
sudo du -sh /var/lib/docker/volumes/jaeger-deployment_jaeger-badger-data/_data 2>/dev/null || echo "Volume not accessible"
echo ""

echo "=== Badger Detailed Breakdown ==="
sudo du -sh /var/lib/docker/volumes/jaeger-deployment_jaeger-badger-data/_data/* 2>/dev/null | sort -rh || echo "No data yet"
echo ""

echo "=== Docker Container Stats ==="
sudo docker stats --no-stream jaeger-prod --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}"
echo ""

echo "=== Recent Activity (last 10 log lines) ==="
sudo docker logs jaeger-prod --tail 10 2>&1
