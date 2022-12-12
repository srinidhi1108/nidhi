FROM --platform=linux/amd64 alpine:3.16
MAINTAINER mlabouardy <mohamed@tailwarden.com>

ENV VERSION 3.0.0

RUN apk update && apk add curl
RUN curl -L https://cli.komiser.io/$VERSION/linux/komiser -o /usr/bin/komiser && \
    chmod +x /usr/bin/komiser && \
    mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

EXPOSE $PORT
ENTRYPOINT ["komiser", "start"]