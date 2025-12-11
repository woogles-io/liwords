#!/bin/bash
set -e

echo "=== Installing Docker and Docker Compose on EC2 (Amazon Linux 2023) ==="

# Update system
echo "Updating system packages..."
sudo yum update -y

# Install Docker
echo "Installing Docker..."
sudo yum install -y docker

# Start Docker service
echo "Starting Docker service..."
sudo systemctl start docker
sudo systemctl enable docker

# Add ec2-user to docker group
echo "Adding current user to docker group..."
sudo usermod -a -G docker ec2-user

# Install Docker Compose
echo "Installing Docker Compose..."
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Verify installations
echo ""
echo "=== Verifying Installation ==="
docker --version
docker-compose --version

echo ""
echo "=== Installation Complete! ==="
echo "IMPORTANT: Log out and log back in for docker group changes to take effect"
echo "Or run: newgrp docker"
