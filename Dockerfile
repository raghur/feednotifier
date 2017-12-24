FROM alpine:3.7
RUN apk add --update ca-certificates libxslt
RUN mkdir -p /app/assets
WORKDIR /app
COPY feednotifier /app/
COPY assets /app/assets/
VOLUME ["/data"]
CMD ["./feednotifier", "-l", "debug", "-t", "$pushover", "/data/watchfile.txt"]

