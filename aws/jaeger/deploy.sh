#!/bin/bash
set -e

echo "=== Deploying Jaeger v2 with Badger Storage ==="

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "ERROR: Docker is not running or you don't have permissions"
    echo "Try: newgrp docker"
    exit 1
fi

# Pull the latest image
echo "Pulling Jaeger v2 image..."
docker-compose pull

# Stop existing container if running
echo "Stopping existing Jaeger container (if any)..."
docker-compose down || true

# Start Jaeger
echo "Starting Jaeger v2..."
docker-compose up -d

# Wait for health check
echo "Waiting for Jaeger to be healthy..."
sleep 10

# Check status
echo ""
echo "=== Deployment Status ==="
docker-compose ps

echo ""
echo "=== Checking Logs ==="
docker-compose logs --tail=20

echo ""
echo "=== Deployment Complete! ==="
echo "Jaeger UI: http://$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4):16686"
echo ""
echo "Useful commands:"
echo "  View logs: docker-compose logs -f"
echo "  Restart: docker-compose restart"
echo "  Stop: docker-compose down"
echo "  Check disk usage: docker system df -v | grep jaeger"
