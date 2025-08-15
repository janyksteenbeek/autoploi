FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build go build -o /bin/autoploi ./cmd/autoploi

FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl
COPY --from=build /bin/autoploi /autoploi
ENTRYPOINT ["/autoploi"]
