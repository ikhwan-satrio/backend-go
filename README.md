# backend golang

## dir structure
```
.
├── cmd
│   ├── root.go
│   └── watch.go
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── internal
│   ├── app
│   │   ├── bootstrap.go
│   │   ├── controller
│   │   │   └── auth.go
│   │   ├── database
│   │   │   ├── db.go
│   │   │   ├── migrate.go
│   │   │   └── models.go
│   │   └── router.go
│   ├── auth
│   │   └── session.go
│   ├── config
│   │   └── config.go
│   ├── dto
│   │   ├── login
│   │   │   ├── req.go
│   │   │   └── res.go
│   │   └── users
│   │       └── create.go
│   ├── httpx
│   │   └── json.go
│   └── middleware
│       └── middleware.go
├── LICENSE
├── main.go
├── README.md
├── secretspec.toml
├── target
│   └── backend-go
└── tmp
    ├── build-errors.log
    └── main


```

## build and running apps

### using local
```sh
git clone https://github.com/ikhwan-satrio/backend-go
cd ./backend-go
go build -o ./target/backend-go
./target/backend-go
```

### using docker
```sh
git clone https://github.com/ikhwan-satrio/backend-go
cd ./backend-go

docker-compose up -d
```
