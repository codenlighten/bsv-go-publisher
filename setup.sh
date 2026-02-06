#!/bin/bash

# BSV AKUA Broadcast Server - Setup Wizard
# Interactive setup for first-time users

set -e

BOLD='\033[1m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

clear
echo -e "${BOLD}${BLUE}"
echo "╔════════════════════════════════════════════════════╗"
echo "║   BSV AKUA Broadcast Server - Setup Wizard        ║"
echo "╚════════════════════════════════════════════════════╝"
echo -e "${NC}\n"

# Check if .env already exists
if [ -f .env ]; then
  echo -e "${YELLOW}⚠️  .env file already exists!${NC}"
  read -p "Do you want to overwrite it? (y/N): " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Setup cancelled."
    exit 0
  fi
  mv .env .env.backup.$(date +%Y%m%d-%H%M%S)
  echo -e "${GREEN}✓ Backed up existing .env${NC}\n"
fi

# Copy template
cp .env.example .env
echo -e "${GREEN}✓ Created .env from template${NC}\n"

# MongoDB Password
echo -e "${BOLD}1. MongoDB Configuration${NC}"
echo "Enter a secure password for MongoDB:"
read -s MONGO_PASSWORD
echo
if [ -z "$MONGO_PASSWORD" ]; then
  MONGO_PASSWORD="bsv_broadcast_$(openssl rand -hex 8)"
  echo -e "${YELLOW}Using generated password${NC}"
fi
sed -i "s/your_secure_mongo_password_here/$MONGO_PASSWORD/" .env
echo -e "${GREEN}✓ MongoDB password set${NC}\n"

# BSV Network
echo -e "${BOLD}2. BSV Network${NC}"
echo "Select network:"
echo "  1) mainnet (default)"
echo "  2) testnet"
echo "  3) regtest (local development)"
read -p "Choice [1-3] (default: 1): " NETWORK_CHOICE

case $NETWORK_CHOICE in
  2)
    sed -i "s/BSV_NETWORK=mainnet/BSV_NETWORK=testnet/" .env
    echo -e "${GREEN}✓ Using testnet${NC}\n"
    ;;
  3)
    sed -i "s/BSV_NETWORK=mainnet/BSV_NETWORK=regtest/" .env
    echo -e "${GREEN}✓ Using regtest${NC}\n"
    ;;
  *)
    echo -e "${GREEN}✓ Using mainnet${NC}\n"
    ;;
esac

# ARC Configuration
echo -e "${BOLD}3. ARC Configuration${NC}"
echo "Select ARC provider:"
echo "  1) GorillaPool (default)"
echo "  2) TAAL"
echo "  3) Custom URL"
read -p "Choice [1-3] (default: 1): " ARC_CHOICE

case $ARC_CHOICE in
  2)
    sed -i "s|ARC_URL=https://arc.gorillapool.io|ARC_URL=https://api.taal.com/arc|" .env
    echo "Using TAAL ARC"
    ;;
  3)
    read -p "Enter custom ARC URL: " CUSTOM_URL
    sed -i "s|ARC_URL=https://arc.gorillapool.io|ARC_URL=$CUSTOM_URL|" .env
    echo "Using custom ARC: $CUSTOM_URL"
    ;;
  *)
    echo "Using GorillaPool ARC"
    ;;
esac

read -p "Enter your ARC API token: " ARC_TOKEN
if [ -z "$ARC_TOKEN" ]; then
  echo -e "${YELLOW}⚠️  No ARC token provided. You'll need to add it later.${NC}"
else
  sed -i "s/your_arc_api_token_here/$ARC_TOKEN/" .env
  echo -e "${GREEN}✓ ARC token configured${NC}"
fi
echo ""

# Train Configuration
echo -e "${BOLD}4. Train Configuration${NC}"
echo "How many seconds between train departures?"
read -p "Interval in seconds (default: 3): " TRAIN_INTERVAL
if [ -n "$TRAIN_INTERVAL" ]; then
  sed -i "s/TRAIN_INTERVAL=3s/TRAIN_INTERVAL=${TRAIN_INTERVAL}s/" .env
fi

echo "Maximum transactions per batch?"
read -p "Max batch size (default: 1000): " TRAIN_MAX
if [ -n "$TRAIN_MAX" ]; then
  sed -i "s/TRAIN_MAX_BATCH=1000/TRAIN_MAX_BATCH=$TRAIN_MAX/" .env
fi
echo -e "${GREEN}✓ Train configuration set${NC}\n"

# Summary
echo -e "${BOLD}${GREEN}═══════════════════════════════════════════════${NC}"
echo -e "${BOLD}${GREEN}     Configuration Complete!${NC}"
echo -e "${BOLD}${GREEN}═══════════════════════════════════════════════${NC}\n"

echo "Your server is ready to start. Here's what will happen:"
echo ""
echo "  1. Server will generate BSV keypairs on first run"
echo "  2. You'll need to copy those keys back into .env"
echo "  3. Fund the funding address with BSV"
echo "  4. Create publishing UTXOs (splitter)"
echo "  5. Start broadcasting!"
echo ""

read -p "Do you want to start the server now? (Y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Nn]$ ]]; then
  echo ""
  echo "To start later, run:"
  echo -e "  ${BOLD}make run${NC}"
  exit 0
fi

echo ""
echo -e "${BOLD}Starting server...${NC}\n"

# Check Docker
if ! command -v docker &> /dev/null; then
  echo -e "${RED}✗ Docker not found. Please install Docker first.${NC}"
  exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
  echo -e "${RED}✗ Docker Compose not found. Please install Docker Compose first.${NC}"
  exit 1
fi

# Start services
docker-compose up -d

echo ""
echo -e "${GREEN}✓ Server started!${NC}\n"
echo "View logs with: ${BOLD}make logs${NC}"
echo ""
echo -e "${YELLOW}IMPORTANT:${NC}"
echo "  The server will generate keypairs on first startup."
echo "  Check the logs for your funding and publishing addresses:"
echo -e "    ${BOLD}make logs${NC}"
echo ""
echo "  Then:"
echo "  1. Copy the private keys to your .env file"
echo "  2. Restart: ${BOLD}make restart${NC}"
echo "  3. Fund your address with BSV"
echo ""
echo "Need help? See ${BOLD}QUICKSTART.md${NC}"
