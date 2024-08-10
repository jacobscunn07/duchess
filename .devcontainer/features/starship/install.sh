#!/bin/sh
set -e

if [ -z "${VERSION}" ]; then
  exit 1
else
  curl -sS https://raw.githubusercontent.com/starship/starship/${VERSION}/install/install.sh | sh -s -- --yes
fi

mkdir -p ${_REMOTE_USER_HOME}/.config && echo 'eval "$(starship init zsh)"' >> ${_REMOTE_USER_HOME}/.zshrc

if [ -n "${PRESET}" ]; then
  starship preset ${PRESET} -o ${_REMOTE_USER_HOME}/.config/starship.toml
fi
