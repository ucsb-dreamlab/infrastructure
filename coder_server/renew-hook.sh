#!/bin/sh
# /etc/letsencrypt/renewal-hooks/deploy/coder.sh
certsdir="/etc/coder.d"
domain="coder.chaparral.io"
basedomain="$(basename $RENEWED_LINEAGE)"
youruser="coder"
yourgroup="$(id -ng $youruser)"

if [ "$domain" = "$basedomain" ];then
    cp "$RENEWED_LINEAGE/fullchain.pem" "$certsdir/fullchain.pem"
    cp "$RENEWED_LINEAGE/privkey.pem" "$certsdir/privkey.pem"
    chown $youruser:$yourgroup "$certsdir/fullchain.pem"
    chown $youruser:$yourgroup "$certsdir/privkey.pem"
	systemctl restart coder
fi
