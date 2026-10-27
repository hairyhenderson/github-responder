FROM golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS build

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

FROM alpine:3.24@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a AS alpine

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
