FROM golang:1.23-bookworm

RUN apt-get update \
    && apt-get install -y --no-install-recommends docker.io ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY . /app
RUN /usr/local/go/bin/go build -o /usr/local/bin/kkg-sandbox .

ENV SANDBOX_ADDR=:8082
ENV SANDBOX_GO_MODE=container
ENV SANDBOX_RUNNER_IMAGE=golang:1.23-bookworm

EXPOSE 8082

CMD ["/usr/local/bin/kkg-sandbox"]
