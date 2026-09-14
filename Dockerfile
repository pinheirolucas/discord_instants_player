ARG VERSION=dev

FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN apk add --no-cache ca-certificates

ARG VERSION
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
    -ldflags "-s -w -X github.com/pinheirolucas/discord_instants_player/cmd.Version=${VERSION}" \
    -o /out/discord_instants_player .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/discord_instants_player /discord_instants_player

ENV HOME=/home/app
USER 65532:65532

VOLUME /home/app/.instants
EXPOSE 9001

ENTRYPOINT ["/discord_instants_player"]
