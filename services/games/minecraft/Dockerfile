FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl jq unzip gosu \
    openjdk-17-jre-headless \
  && rm -rf /var/lib/apt/lists/*

RUN useradd -m -u 1000 minecraft
WORKDIR /opt/minecraft

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

VOLUME ["/data"]

EXPOSE 25565 25575
ENTRYPOINT ["/entrypoint.sh"]
