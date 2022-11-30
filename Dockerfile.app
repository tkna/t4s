FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

# Copy the go source
COPY app/app.go app.go
COPY api/ api/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app app.go

FROM scratch as runner
COPY --from=builder /workspace/app /app
COPY app/static/ /static/
USER 65532:65532
ENTRYPOINT [ "/app" ]
