FROM golang:1.26-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN mkdir -p internals/resources/banners internals/resources/profiles

COPY . .
RUN go build -o main .

EXPOSE 8080
CMD ["./main"]
