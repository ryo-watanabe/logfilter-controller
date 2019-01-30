FROM alpine
RUN apk add curl coreutils
COPY entrypoint.sh /
RUN chmod +x /es-housekeep.sh
ENTRYPOINT ["/es-housekeep.sh"]
