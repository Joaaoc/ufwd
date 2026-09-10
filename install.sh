#!/bin/bash
set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Starting UFWD Installation...${NC}"

# 1. Root Check
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}Error: Please run this script as root (use sudo).${NC}"
  exit 1
fi

# 2. Dependency Check
DEPENDENCIES=("curl" "tar" "ufw" "iptables")
for cmd in "${DEPENDENCIES[@]}"; do
  if ! command -v "$cmd" &> /dev/null; then
    echo -e "${RED}Error: Required command '$cmd' is not installed.${NC}"
    echo -e "${YELLOW}Please install it using your system's package manager (apt, yum, pacman) and try again.${NC}"
    exit 1
  fi
done

# 3. Architecture Detection
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)
    FILE_NAME="ufwd_linux_amd64.tar.gz"
    ;;
  aarch64|arm64)
    FILE_NAME="ufwd_linux_arm64.tar.gz"
    ;;
  *)
    echo -e "${RED}Error: Unsupported architecture '$ARCH'.${NC}"
    echo -e "${YELLOW}UFWD currently supports x86_64 (amd64) and aarch64 (arm64).${NC}"
    exit 1
    ;;
esac

DOWNLOAD_URL="https://github.com/Joaaoc/ufwd/releases/latest/download/${FILE_NAME}"

# 4. Download and Install
echo -e "Detected architecture: ${YELLOW}$ARCH${NC}"
echo "Downloading latest release..."

TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

if ! curl -sL "$DOWNLOAD_URL" -o "$FILE_NAME"; then
  echo -e "${RED}Error: Failed to download UFWD from GitHub.${NC}"
  exit 1
fi

echo "Extracting..."
tar -xzf "$FILE_NAME"

if [ ! -f "ufwd" ]; then
  echo -e "${RED}Error: Binary not found in the downloaded archive.${NC}"
  exit 1
fi

echo "Installing to /usr/local/bin/ufwd..."
mv ufwd /usr/local/bin/ufwd
chmod +x /usr/local/bin/ufwd

# Clean up
cd /
rm -rf "$TEMP_DIR"

# 5. Run UFWD setup
echo -e "\n${GREEN}Download complete. Running internal UFWD setup...${NC}"
/usr/local/bin/ufwd install --force
echo "Installation completed successfully"
