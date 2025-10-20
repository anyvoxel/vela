FROM mcr.microsoft.com/devcontainers/go:1.25-bookworm

RUN sed -i 's|http://deb.debian.org/|http://mirrors.aliyun.com/|g' /etc/apt/sources.list.d/debian.sources \
    && apt-get update \
    && PREFIX="/usr/local" \
    && VERSION="1.51.0" \
    && curl -sSL "https://github.com/bufbuild/buf/releases/download/v${VERSION}/buf-$(uname -s)-$(uname -m).tar.gz" | \
    tar -xvzf - -C "${PREFIX}" --strip-components 1 \
    && rm -rf /var/lib/apt/lists/* \
    && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.4.0