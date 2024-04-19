#!/bin/sh
litestream restore -if-db-not-exists -if-replica-exists /tmp/chaparral.sqlite3 && \
litestream replicate -exec "chaparral -c /etc/chaparral.yml"
