# Copyright IBM Corp. 2020, 2025
# SPDX-License-Identifier: MPL-2.0

FROM golang:1.20

WORKDIR /go/src/terraform-provider-boundary
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...
