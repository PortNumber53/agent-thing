FROM alpine:3.20

RUN apk add --no-cache bash coreutils git openssh

RUN adduser -D -u 1000 developer
USER developer
WORKDIR /home/developer

CMD ["tail", "-f", "/dev/null"]
