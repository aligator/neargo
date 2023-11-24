FROM golang:1.21-alpine AS builder

COPY go.mod go.sum /app/
WORKDIR /app

RUN go get -v -d ./...

COPY . /app/

RUN go build .

# Final stage only containing the binary
FROM alpine AS final

LABEL org.label-schema.schema-version="1.0"
LABEL org.label-schema.name="neargo"
LABEL org.label-schema.description="A simple postal code proximity search webservice"

COPY --from=builder /app/neargo /bin

WORKDIR /data

# You may use --geonames-file geonames.zip together with a volume to /data to store a specific zip.
VOLUME [ "/data" ]
EXPOSE 3141

ENTRYPOINT ["neargo"]