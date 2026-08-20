# syntax=docker/dockerfile:1
FROM golang:1.26.5-alpine AS build
WORKDIR /source

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/worker ./cmd/worker

FROM gcr.io/distroless/static-debian13:nonroot AS runtime

COPY --from=build /out/worker /worker

USER nonroot:nonroot
ENTRYPOINT ["/worker"]
