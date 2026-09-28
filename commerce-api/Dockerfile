ARG GO_VERSION=1.26.4

FROM golang:${GO_VERSION}-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN set -eux; \
    mkdir -p /out; \
    for dir in ./cmd/*; do \
        [ -d "$dir" ] || continue; \
        name="$(basename "$dir")"; \
        CGO_ENABLED=0 GOOS=linux go build \
            -buildvcs=false \
            -trimpath \
            -ldflags='-s -w' \
            -o "/out/${name}" \
            "$dir"; \
    done

FROM debian:bookworm-slim AS runtime

RUN set -eux; \
    apt-get update; \
    apt-get install -y --no-install-recommends ca-certificates curl; \
    rm -rf /var/lib/apt/lists/*; \
    groupadd --system --gid 10001 commerce; \
    useradd --system --uid 10001 --gid commerce --create-home --home-dir /home/commerce commerce

WORKDIR /app

COPY --from=build --chown=commerce:commerce /out/ /app/bin/

ENV PATH="/app/bin:${PATH}"

USER 10001:10001

EXPOSE 8080

STOPSIGNAL SIGTERM

CMD ["/app/bin/api"]