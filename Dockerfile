FROM golang:1.27

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o ./backend-go

EXPOSE 3000
CMD ["./backend-go","serve"]
