#!/bin/sh
# Copy tvhtc2-client to /srv/tvhtc2, which should be a volume shared
# between this container and the tvheadend container. This will allow
# tvheadend to access the tvhtc2-client binary (along with the socket).
cp /usr/bin/tvhtc2-client /srv/tvhtc2/tvhtc2-client

# Start the server
/usr/bin/tvhtc2