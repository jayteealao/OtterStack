#!/bin/bash
set -e  # Exit on error

echo "=== Aperture Deployment Script ==="
echo "This script will:"
echo "1. Update OtterStack to latest version"
echo "2. Build and install OtterStack"
echo "3. Deploy Aperture"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Update OtterStack
echo -e "${YELLOW}Step 1: Updating OtterStack...${NC}"
cd ~/OtterStack
git fetch origin
git pull origin feat/stream-docker-output

# Step 2: Build OtterStack
echo -e "${YELLOW}Step 2: Building OtterStack...${NC}"
go build -o otterstack .

# Step 3: Install OtterStack
echo -e "${YELLOW}Step 3: Installing OtterStack...${NC}"
sudo cp otterstack /usr/local/bin/otterstack
sudo chmod +x /usr/local/bin/otterstack

# Verify installation
echo -e "${YELLOW}Verifying installation...${NC}"
which otterstack
otterstack version

# Step 4: Deploy Aperture
echo -e "${YELLOW}Step 4: Deploying Aperture (032e3b4)...${NC}"
echo "This will show container logs if it fails..."
otterstack deploy aperture 032e3b4 -v

# If we got here, deployment succeeded
echo -e "${GREEN}=== Deployment Successful! ===${NC}"
echo ""
echo "Check status:"
echo "  docker ps | grep aperture"
echo ""
echo "Check logs:"
echo "  docker logs aperture-032e3b4-aperture-gateway-1"
echo "  docker logs aperture-032e3b4-aperture-web-1"
