#!/usr/bin/env bash
set -e

if [[ "${ELEVATED_USER}" == "true" ]]; then
  exec "$@"
else
  exec gosu "${USER_NAME}:${GROUP_NAME}" "$@"
fi
