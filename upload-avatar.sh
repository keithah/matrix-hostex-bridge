#!/bin/bash

# Script to upload Hostex logo and get mxc:// URI
# This will be run when the bridge is running to get the proper avatar URI

echo "To upload the Hostex logo avatar:"
echo "1. Start your bridge: ./mautrix-hostex"
echo "2. In another terminal, run this curl command:"
echo ""
echo "curl -X POST \\"
echo "  -H 'Authorization: Bearer hua_QHZND3dMVOa6otDyIgkkZmz2g91uriCLR0NbBcVh2No1uP4ICcBO5_0PNkzb' \\"
echo "  -H 'Content-Type: image/png' \\"
echo "  --data-binary '@hostex-logo.png' \\"
echo "  'https://matrix.beeper.com/_hungryserv/keithah/_matrix/media/v3/upload?filename=hostex-logo.png&user_id=%40sh-hostexbot%3Abeeper.local'"
echo ""
echo "3. Copy the mxc:// URI from the response"
echo "4. Update config.yaml with the new avatar URI"