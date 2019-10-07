# Build storage provisioner binary
FROM golang:1.12.5 as builder

WORKDIR /go/src/github.com/mayadata-io/storage-provisioner

# copy go modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# ensure vendoring is up-to-date by running make vendor in your local
# setup
#
# we cache the vendored dependencies before building and copying source
# so that we don't need to re-download when source changes don't invalidate
# our downloaded layer
RUN GO111MODULE=on go mod download
RUN GO111MODULE=on go mod vendor

# copy build manifests
COPY Makefile Makefile

# copy source files
COPY build/ build/
COPY cmd/ cmd/
COPY hack/ hack/
COPY pkg/ pkg/
COPY storage/ storage/

# build storage provisioner binary
RUN make

# Use distroless as minimal base image to package the final binary
FROM gcr.io/distroless/static:latest
LABEL maintainers="MayaData Authors"
LABEL description="DAO Storage Provisioner"

COPY --from=builder /go/src/github.com/mayadata-io/storage-provisioner/dao-storprovisioner dao-storprovisioner
ENTRYPOINT ["/dao-storprovisioner"]
