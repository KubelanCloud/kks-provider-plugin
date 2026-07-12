FROM golang:1.24-alpine AS build

WORKDIR /build
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /kks-provider .

FROM alpine:3.21
RUN apk add --no-cache mount util-linux e2fsprogs findmnt
COPY --from=build /kks-provider /kks-provider
USER 0:0
ENTRYPOINT ["/kks-provider"]
