FROM golang:1.25 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -trimpath -ldflags "-s -w -X webhookd/internal/buildinfo.Version=${VERSION}" \
	-o /out/webhookd ./cmd/webhookd

FROM gcr.io/distroless/base-debian12:nonroot

COPY --from=build /out/webhookd /webhookd

EXPOSE 1337

USER nonroot:nonroot
ENTRYPOINT ["/webhookd"]
