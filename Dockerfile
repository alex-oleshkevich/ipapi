FROM golang:1.23 AS builder
SHELL ["/bin/bash", "-c"]
ARG GIT_COMMIT=unspecified
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o ipapi

FROM alpine:latest
ARG GIT_COMMIT=unspecified
WORKDIR /app
COPY --from=builder /app/ipapi /app/ipapi
ADD data/ /data/
EXPOSE 8080
RUN echo $ARG GIT_COMMIT > /app/commithash
ENV GEOIP_DB_PATH=/data/GeoLite2-City.mmdb
CMD ["/app/ipapi"]
