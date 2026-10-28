FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# docker:cli ships the docker CLI and the compose plugin.
FROM docker:29-cli
COPY --from=build /out/server /server
ENV LISTEN_ADDR=:8080 STACKS_DIR=/opt/stacks DATA_DIR=/data
EXPOSE 8080
ENTRYPOINT ["/server"]
