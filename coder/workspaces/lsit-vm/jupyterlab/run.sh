#!/usr/bin/env bash

UV="$HOME/.local/bin/uv"

# install uv
if ! command -v $UV > /dev/null 2>&1; then
   curl -LsSf https://astral.sh/uv/install.sh | sh
fi

if [ -n "${BASE_URL}" ]; then
  BASE_URL_FLAG="--ServerApp.base_url=${BASE_URL}"
fi

BOLD='\033[0;1m'

# check if jupyterlab is installed
if ! command -v jupyter-lab > /dev/null 2>&1; then
  # install jupyterlab
  printf "$${BOLD}Installing jupyterlab in .venv!\n"
  $UV venv
  $UV pip install -q jupyterlab && printf "%s\n" "🥳 jupyterlab has been installed"
else
  printf "%s\n\n" "🥳 jupyterlab is already installed"
fi

printf "👷 Starting jupyterlab in background..."
printf "check logs at ${LOG_PATH}"
$UV run jupyter-lab --no-browser \
  "$BASE_URL_FLAG" \
  --ServerApp.ip='*' \
  --ServerApp.port="${PORT}" \
  --ServerApp.token='' \
  --ServerApp.password='' \
  > "${LOG_PATH}" 2>&1 &
