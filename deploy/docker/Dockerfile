# Multi-stage static build (no secrets in image).
FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X go-imapsync/cmd.version=${VERSION}" -o /out/go-imapsync .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/go-imapsync /go-imapsync
USER nonroot:nonroot
ENTRYPOINT ["/go-imapsync"]
