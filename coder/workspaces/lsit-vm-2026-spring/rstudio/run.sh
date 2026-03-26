#!/bin/env bash
set -e

# store coder agent variables for rstudio session
printenv | grep '^CODER_\|^GIT_\|^SSH_' > "$HOME/.Renviron"

sudo systemctl enable --now rstudio-server
