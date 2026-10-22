FROM golang:1.27.3-alpine@sha256:b8c9b82affbbb82a020bbcf998dad2ba9516944ac7e883524642a481a9505e9c AS build

RUN apk add --no-cache make git

RUN mkdir -p /go/src/github.com/hairyhenderson/github-responder
WORKDIR /go/src/github.com/hairyhenderson/github-responder
COPY . /go/src/github.com/hairyhenderson/github-responder

ARG VCS_REF
ARG VERSION
ARG CODEOWNERS

RUN make build-x

FROM scratch AS artifacts

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /go/src/github.com/hairyhenderson/github-responder/bin/* /bin/

CMD [ "/bin/github-responder_linux-amd64" ]

FROM scratch AS latest

ARG OS=linux
ARG ARCH=amd64

COPY --from=artifacts /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=artifacts /bin/github-responder_${OS}-${ARCH} /github-responder

ARG VCS_REF
ARG VERSION
ARG CODEOWNERS

LABEL org.opencontainers.image.revision=$VCS_REF \
      org.opencontainers.image.title=github-responder \
      org.opencontainers.image.authors=$CODEOWNERS \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.source="https://github.com/hairyhenderson/github-responder"

ENTRYPOINT [ "/github-responder" ]

FROM alpine:3.24@sha256:71e9b7bae160ba167fa17414401b94cbcdaf3c32feeab9298bd794a2a0614c35 AS alpine

ARG OS=linux
ARG ARCH=amd64

RUN apk add --no-cache ca-certificates
COPY --from=artifacts /bin/github-responder_${OS}-${ARCH} /bin/github-responder

ARG VCS_REF
ARG VERSION
ARG CODEOWNERS

LABEL org.opencontainers.image.revision=$VCS_REF \
      org.opencontainers.image.title=github-responder \
      org.opencontainers.image.authors=$CODEOWNERS \
      org.opencontainers.image.version=$VERSION \
      org.opencontainers.image.source="https://github.com/hairyhenderson/github-responder"

ENTRYPOINT [ "/bin/github-responder" ]
