FROM golang:1.27.2-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN build_version= && \
    if [ -f RELEASE_VERSION ]; then build_version="$(cat RELEASE_VERSION)"; fi && \
    { [ -z "$build_version" ] || printf '%s\n' "$build_version" | LC_ALL=C grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$'; } && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=$build_version" -o /out/myjournal ./cmd/app

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/myjournal /myjournal
USER 65532:65532
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/myjournal"]
