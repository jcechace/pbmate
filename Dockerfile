FROM alpine:3.21
ARG TARGETARCH
WORKDIR /env
COPY pbmate_linux_${TARGETARCH} /usr/local/bin/pbmate
ENTRYPOINT ["pbmate"]
