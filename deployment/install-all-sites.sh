#!/bin/bash

#############################################
# Install All 3 News Websites
# Usage: ./install-all-sites.sh [email]
#############################################

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}"
echo "=============================================="
echo "   Multi-Site News Website Installer"
echo "=============================================="
echo -e "${NC}"

EMAIL=${1:-"admin@example.com"}

echo "This script will install 3 independent news websites:"
echo ""
echo "  1. enginosys.com    → Port 8081"
echo "  2. cryptonlisys.com → Port 8082"  
echo "  3. technolisys.com  → Port 8083"
echo ""
echo "Email for all sites: $EMAIL"
echo ""
read -p "Continue? (y/N): " confirm

if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
    echo "Cancelled."
    exit 0
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo ""
echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Installing Site 1: enginosys.com${NC}"
echo -e "${YELLOW}========================================${NC}"
bash "$SCRIPT_DIR/multi-site-install.sh" enginosys.com 1 "$EMAIL"

echo ""
echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Installing Site 2: cryptonlisys.com${NC}"
echo -e "${YELLOW}========================================${NC}"
bash "$SCRIPT_DIR/multi-site-install.sh" cryptonlisys.com 2 "$EMAIL"

echo ""
echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}Installing Site 3: technolisys.com${NC}"
echo -e "${YELLOW}========================================${NC}"
bash "$SCRIPT_DIR/multi-site-install.sh" technolisys.com 3 "$EMAIL"

echo ""
echo -e "${GREEN}=============================================="
echo "   All Sites Installed Successfully!"
echo "==============================================${NC}"
echo ""
echo "Site 1: https://enginosys.com"
echo "        Credentials: /opt/enginosys-website/CREDENTIALS.txt"
echo ""
echo "Site 2: https://cryptonlisys.com"
echo "        Credentials: /opt/cryptonlisys-website/CREDENTIALS.txt"
echo ""
echo "Site 3: https://technolisys.com"
echo "        Credentials: /opt/technolisys-website/CREDENTIALS.txt"
echo ""
echo "=============================================="
