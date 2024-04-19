FROM cgr.dev/chainguard/busybox:latest
COPY --from=srerickson/chaparral:latest /ko-app/chaparral /usr/local/bin/chaparral
COPY --from=litestream/litestream:latest /usr/local/bin/litestream /usr/local/bin/litestream
COPY chaparral.yml /etc/chaparral.yml
COPY litestream.yml /etc/litestream.yml
COPY start.sh /start.sh
ENV CHAPARRAL_BACKEND=s3://ocfl?region=us-west-2
CMD ["sh","/start.sh"]
