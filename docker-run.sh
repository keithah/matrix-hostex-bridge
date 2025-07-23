#!/bin/bash

cd /data

if [[ ! -f config.yaml ]]; then
    /usr/bin/mautrix-hostex -g -c config.yaml -r registration.yaml
    echo "Generated config files. Please edit config.yaml and restart the container."
    exit 1
fi

exec su-exec $UID:$GID /usr/bin/mautrix-hostex