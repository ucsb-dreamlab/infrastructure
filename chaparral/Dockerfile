FROM srerickson/chaparral:latest
COPY config.yaml config.yaml
ENV CHAPARRAL_BACKEND=s3://ocfl?region=us-west-2
CMD ["/chaparral", "-c", "config.yaml"]
