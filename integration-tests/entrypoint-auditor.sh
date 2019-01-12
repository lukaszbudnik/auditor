#!/bin/sh

DEFAULT_AUDITOR_CONFIG_FILE=".env"

# if auditor config file is not provided explicitly fallback to default one
if [ -z "$AUDITOR_CONFIG_FILE" ]; then
  AUDITOR_CONFIG_FILE=$DEFAULT_AUDITOR_CONFIG_FILE
fi

auditor -configFile "$AUDITOR_CONFIG_FILE"
