#!/bin/env bash
set -e

# store coder agent variables for rstudio session
printenv | grep '^CODER_\|^GIT_\|^SSH_' > "$HOME/.Renviron"

# rsessions shouldn't store coder's env variables between sessions
sudo sh -c 'echo "session-ephemeral-env-vars=GIT_SSH_COMMAND:CODER_AGENT_TOKEN:CODER_SCRIPT_DATA_DIR" > /etc/rstudio/rsession.conf'

# set default user for rstudio-server and rsession to 'coder'
sudo sh -c 'sed -i "s/ubuntu/coder/g" /etc/default/rstudio-server'
sudo sh -c 'sed -i "s/ubuntu/coder/g" /etc/rstudio/rserver.conf'

sudo systemctl enable --now rstudio-server
