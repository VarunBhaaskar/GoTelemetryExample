# syntax=docker/dockerfile:1
# A sample microservice in Go packaged into a container image. 
# This is a sample application to test the capapbilities of Opentelemtry and grafana observability stack

FROM golang:1.20

WORKDIR /app

COPY ./* ./

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /docker_GoTelemetryExample

EXPOSE 8080

CMD ["/docker_GoTelemetryExample"]