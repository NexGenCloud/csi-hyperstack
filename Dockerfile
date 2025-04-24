# syntax=docker/dockerfile:1.4

FROM golang:1.22 as build

ENV TZ=UTC
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

WORKDIR /build

RUN go install github.com/go-task/task/v3/cmd/task@latest
RUN go install github.com/goreleaser/goreleaser@latest

# Install dependencies
COPY --link go.mod go.sum ./

RUN go mod tidy

# Build binary
COPY --link Taskfile.yaml .goreleaser.yaml ./
COPY --link main.go ./
COPY --link pkg ./pkg
# For goreleaser, TODO: remove
COPY --link .git ./.git

RUN task build


FROM alpine:3.20 as runtime

RUN apk add --no-cache --update \
  ca-certificates \
  bash \
  vim \
  curl

RUN curl -o /usr/local/bin/gosu \
      -fsSL "https://github.com/tianon/gosu/releases/download/1.12/gosu-amd64" \
 && chmod +x /usr/local/bin/gosu

RUN curl -fsSL "https://github.com/fullstorydev/grpcurl/releases/download/v1.7.0/grpcurl_1.7.0_linux_x86_64.tar.gz" -o grpcurl_1.7.0_linux_x86_64.tar.gz \
 && tar -xvf grpcurl_1.7.0_linux_x86_64.tar.gz \
 && chmod +x grpcurl

ENV HOME "/app"
WORKDIR "${HOME}"

ENV ELEVATED_USER "false"
ENV GROUP_ID 1000
ENV GROUP_NAME app
ENV USER_ID 1000
ENV USER_NAME app
RUN addgroup -g "${GROUP_ID}" "${GROUP_NAME}" \
 && adduser -u "${USER_ID}" -G "${GROUP_NAME}" -h "${HOME}" -D "${USER_NAME}"

COPY --link build/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

COPY --link --from=build /build/dist/csi-hyperstack /usr/local/bin
RUN chmod +x /usr/local/bin/csi-hyperstack

ENV LOCAL_HEALTH_PORT "8080"
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=1 \
  CMD curl -f "http://localhost:${LOCAL_HEALTH_PORT}/health" || exit 1

ENTRYPOINT ["entrypoint.sh"]
CMD ["csi-hyperstack", "--help"]
