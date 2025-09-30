#!/bin/env bash
set -e

# =============================================================================
# apt packages
# =============================================================================

sudo apt update -y && \
sudo apt upgrade -y && \
sudo apt install -y \
    git \
    unzip \
    nano \
    micro \
    wget \
    curl

# =============================================================================
# Install Docker
# =============================================================================


curl -fsSL https://get.docker.com | sh
sudo systemctl enable docker
sudo usermod -aG docker coder


# =============================================================================
# Install RStudio
# =============================================================================


R_PATH=/usr/bin/R
RSERVER_PATH=/usr/lib/rstudio-server/bin/rserver

RSERVER_INSTALLER=https://download2.rstudio.org/server/jammy/amd64/rstudio-server-2025.09.0-387-amd64.deb
R2U_INSALLER=https://raw.githubusercontent.com/eddelbuettel/r2u/refs/heads/master/inst/scripts/add_cranapt_noble.sh

if [ ! -f "$R_PATH" ]; then
   r2u_script=/tmp/install_R2u.sh
   wget -qO "$r2u_script" "$R2U_INSALLER"
   sudo bash $r2u_script
   rm $r2u_script
fi
