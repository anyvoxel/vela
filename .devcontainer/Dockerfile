FROM --platform=linux/amd64 mcr.microsoft.com/devcontainers/go:1.25-bookworm

RUN sed -i 's|http://deb.debian.org/|http://mirrors.aliyun.com/|g' /etc/apt/sources.list.d/debian.sources \
    && apt-get install -y wget \
    && wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add - \
    && echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google.list \
    && apt-get update \
    && PREFIX="/usr/local" \
    && VERSION="1.51.0" \
    && curl -sSL "https://github.com/bufbuild/buf/releases/download/v${VERSION}/buf-$(uname -s)-$(uname -m).tar.gz" | \
    tar -xvzf - -C "${PREFIX}" --strip-components 1 \
    && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.4.0 \
    && apt-get install -y     wget     gnupg     fonts-liberation     libasound2     libatk-bridge2.0-0     libatk1.0-0     libatspi2.0-0     libcups2     libdbus-1-3     libdrm2     libgbm1     libgtk-3-0     libnspr4     libnss3     libwayland-client0     libxcomposite1     libxdamage1     libxfixes3     libxkbcommon0     libxrandr2     xdg-utils \
    && apt-get install -y google-chrome-stable \
    && curl -fsSL https://deb.nodesource.com/setup_24.x | bash - \
    && apt-get install -y nodejs \
    && npm install -g @google/gemini-cli \
    && rm -rf /var/lib/apt/lists/* \
    && rm -f /etc/apt/sources.list.d/google.list \
    && go install golang.org/x/tools/cmd/goimports@latest \
    && chown -R vscode:golang /go/pkg