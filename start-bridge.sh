#!/bin/bash
# Hostex Bridge Restart Script

BRIDGE_DIR="/Users/keith/src/hostex-bridge-dev"
BRIDGE_CMD="./mautrix-hostex -c config.yaml"
LOG_FILE="$BRIDGE_DIR/bridge-supervisor.log"

cd "$BRIDGE_DIR"

echo "$(date): Starting Hostex Bridge with auto-restart..." | tee -a "$LOG_FILE"

while true; do
    echo "$(date): Starting bridge process..." | tee -a "$LOG_FILE"
    
    # Run the bridge
    $BRIDGE_CMD
    
    # If we get here, the bridge exited
    echo "$(date): Bridge process exited. Restarting in 5 seconds..." | tee -a "$LOG_FILE"
    sleep 5
done