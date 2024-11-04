FROM --platform=${BUILDPLATFORM} golang:alpine as compiler
ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /go/src/pluto
ADD go.mod .
ADD go.sum .
RUN go mod download
COPY . .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o pluto cmd/pluto/main.go

FROM --platform=${TARGETPLATFORM} alpine
COPY --from=compiler /go/src/pluto/pluto /usr/bin/pluto
COPY example/simple/config.yaml /etc/pluto/config.yaml

ENTRYPOINT ["/usr/bin/pluto"]