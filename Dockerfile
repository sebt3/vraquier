FROM golang:1.20-alpine as builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -v -o vraquier

FROM scratch as target
COPY --from=builder /app/vraquier /bin/vraquier
CMD ["/bin/vraquier","--cloud-provider","vraquier","--kubeconfig","/etc/kubernetes/admin.conf"]