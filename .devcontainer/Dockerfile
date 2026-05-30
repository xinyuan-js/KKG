FROM golang:1.23-bookworm AS go

FROM node:20-bookworm

COPY --from=go /usr/local/go /usr/local/go

ENV PATH="/usr/local/go/bin:${PATH}"
