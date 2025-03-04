#!/bin/bash
CONTAINER_NAME=$1
if [ -z "$CONTAINER_NAME" ]; then
    echo "Usage: $0 <container_name>"
    exit 1
fi
docker start "$CONTAINER_NAME"
