#!/usr/bin/env bash

PIXIBIN="$HOME/.pixi/bin"
PIXI="$PIXIBIN/pixi"
JUPYTER="$PIXIBIN/jupyter"

# install pixi
if ! command -v $PIXI > /dev/null 2>&1; then
   curl -fsSL https://pixi.sh/install.sh | sh
   source $HOME/.bashrc
fi

if [ -n "${BASE_URL}" ]; then
  BASE_URL_FLAG="--ServerApp.base_url=${BASE_URL}"
fi

BOLD='\033[0;1m'

# check if jupyterlab is installed
if ! command -v $JUPYTER > /dev/null 2>&1; then
  $PIXI global install jupyter
  $PIXI global install pip
else
  printf "%s\n\n" "🥳 jupyterlab is already installed"
fi

printf "👷 Starting jupyterlab in background..."
printf "check logs at ${LOG_PATH}"
$JUPYTER lab --no-browser \
  "$BASE_URL_FLAG" \
  --ServerApp.ip='*' \
  --ServerApp.port="${PORT}" \
  --ServerApp.token='' \
  --ServerApp.password='' \
  > "${LOG_PATH}" 2>&1 &
