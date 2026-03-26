#!/usr/bin/env bash
set -e

PIXIBIN="$HOME/.pixi/bin"
PIXI="$PIXIBIN/pixi"
JUPYTER="$PIXIBIN/jupyter"

# wait for pixi to become available (up a minute)
PIXI_WAIT=0
until command -v $PIXI > /dev/null 2>&1; do
  if [ $PIXI_WAIT -ge 60 ]; then
    printf "timed out waiting for pixi to become available"
    exit 1
  fi
  echo "waiting for $PIXI..."
  sleep 2
  PIXI_WAIT=$((PIXI_WAIT + 2))
done

if [ -n "${BASE_URL}" ]; then
  BASE_URL_FLAG="--ServerApp.base_url=${BASE_URL}"
fi

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
