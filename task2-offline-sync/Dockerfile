FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /sync-server .

FROM alpine:3
COPY --from=build /sync-server /usr/local/bin/sync-server
EXPOSE 8000
ENTRYPOINT ["sync-server"]
