FROM golang:1.22

WORKDIR /app

# Copy build deps
COPY ./go.mod /app/go.mod
COPY ./go.sum /app/go.sum
RUN go mod download

# Copy the rest of the app & build it
COPY ./ /app
RUN ls -a
RUN mkdir /app/bin
RUN go build -o /app/bin/app /app/cmd/server/main.go
ENV SX_STATIC_DIR=/app/static

CMD ["/app/bin/app"]
