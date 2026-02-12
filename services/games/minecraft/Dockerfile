FROM eclipse-temurin:21-jre-jammy

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl jq unzip gosu \
  && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 minecraft

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

VOLUME ["/data"]
WORKDIR /data

EXPOSE 25565 25575
ENTRYPOINT ["/entrypoint.sh"]
