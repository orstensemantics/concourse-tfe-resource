FROM golang:alpine as build

WORKDIR /go/src/concourse-tfe-resource
COPY . .

RUN go build -o check

FROM alpine

COPY --from=build /go/src/concourse-tfe-resource/check /check

RUN mkdir -p /opt/resource \
 && mv /check /opt/resource \
 && chmod 555 /opt/resource/check \
 && ln -s /opt/resource/check /opt/resource/in \
 && ln -s /opt/resource/check /opt/resource/out
