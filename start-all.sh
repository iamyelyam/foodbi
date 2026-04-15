#!/bin/sh
# Railway production entrypoint. Runs both the API server and the iiko sync
# worker in the same container so one dyno handles everything — no need for
# a separate Railway service.
#
# The sync worker runs in the background; the API server runs in foreground
# so Railway's healthcheck (/health) keeps working. If either process exits,
# the container exits and Railway restarts everything.

set -e

echo "[start-all] booting sync worker in background"
./sync &
SYNC_PID=$!

# When the API server exits, also kill the sync worker so Railway restarts
# cleanly (otherwise we'd have an orphaned sync process).
trap "kill $SYNC_PID 2>/dev/null || true" EXIT

echo "[start-all] starting API server in foreground (PID of sync: $SYNC_PID)"
exec ./api
