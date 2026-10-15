FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build_base

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY ./ ./

ENV CGO_ENABLED=0
ARG TARGETOS TARGETARCH VERSION=dev
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags "-w -s -X main.version=$VERSION" -trimpath -buildvcs=false -o speedtest .

FROM scratch
WORKDIR /app
COPY --from=build_base /build/speedtest ./
COPY settings.toml ./

EXPOSE 8989

CMD ["./speedtest"]
