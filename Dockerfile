FROM golang AS builder

WORKDIR /app
COPY . .

ENV GOFLAGS="-buildvcs=false"
ENV CGO_ENABLED=0
RUN go build -o /usr/local/bin/reeve-step .

FROM alpine

COPY --chmod=755 --from=builder /usr/local/bin/reeve-step /usr/local/bin/

WORKDIR /reeve/src

# API_URL: forgejo API URL
ENV API_URL=
# API_USER: user for authentication
ENV API_USER=
# API_PASSWORD: password for authentication
ENV API_PASSWORD=
# PACKAGE_OWNER: owner of the package
ENV PACKAGE_OWNER=
# PACKAGE_NAME: package name
ENV PACKAGE_NAME=
# PACKAGE_VERSION: package version
ENV PACKAGE_VERSION=
# PACKAGE_REPOSITORY: (optional) repository to which the package is to be linked
ENV PACKAGE_REPOSITORY=
# FILES: Space separated list of file patterns (see https://pkg.go.dev/github.com/bmatcuk/doublestar/v4#Match) to be included (shell syntax)
ENV FILES=

ENTRYPOINT ["reeve-step"]
