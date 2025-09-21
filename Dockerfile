FROM golang:1.24 as build
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod tidy
COPY . .
RUN go mod tidy && go mod download
ARG VERSION
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -ldflags "-s -w \
        -X main.name=csi-hyperstack \
        -X main.version=${VERSION} \
        -X k8s.io/csi-hyperstack/pkg/driver.DriverName=hyperstack.csi.nexgencloud.com \
        -X k8s.io/csi-hyperstack/pkg/driver.DriverVersion=${VERSION}" \
      -o /csi-hyperstack

FROM alpine:3.20 as runtime
RUN apk add --no-cache --update e2fsprogs
RUN wget "https://github.com/fullstorydev/grpcurl/releases/download/v1.7.0/grpcurl_1.7.0_linux_x86_64.tar.gz" \
 && tar -xvf grpcurl_1.7.0_linux_x86_64.tar.gz -C /usr/local/bin grpcurl \
 && chmod +x /usr/local/bin/grpcurl \
 && rm -rf grpcurl_1.7.0_linux_x86_64.tar.gz
COPY --from=build /csi-hyperstack .
RUN chmod +x csi-hyperstack
RUN mkdir -p /csi
ENTRYPOINT ["/csi-hyperstack"]