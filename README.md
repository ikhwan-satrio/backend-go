# backend golang

## Directory structure
```
.
├── cmd
│   ├── root.go
│   └── watch.go
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── internal
│   ├── app
│   │   ├── bootstrap.go
│   │   ├── controller
│   │   │   ├── auth.go
│   │   │   └── oauth.go
│   │   ├── database
│   │   │   ├── db.go
│   │   │   ├── migrate.go
│   │   │   └── models.go
│   │   └── router.go
│   ├── auth
│   │   ├── providers
│   │   │   └── github_oauth.go
│   │   └── session.go
│   ├── config
│   │   └── config.go
│   ├── dto
│   │   ├── login
│   │   │   ├── req.go
│   │   │   └── res.go
│   │   └── users
│   │       └── create.go
│   ├── httpx
│   │   └── json.go
│   └── middleware
│       └── middleware.go
├── LICENSE
├── main.go
├── README.md
├── secretspec.toml
├── target
│   └── backend-go
└── tmp
    ├── build-errors.log
    └── main
```

## Building and running the app

### Using local
```sh
git clone https://github.com/ikhwan-satrio/backend-go
cd ./backend-go
cp .env.example .env
go build -o ./target/backend-go
./target/backend-go serve
```

Run database migrations:
```sh
./target/backend-go migrate
```

### Using Docker
```sh
git clone https://github.com/ikhwan-satrio/backend-go
cd ./backend-go
cp .env.example .env

docker-compose up -d
```

## Environment variables
Copy `.env.example` to `.env` and adjust the values. GitHub OAuth login requires
`GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, and `GITHUB_REDIRECT_URL` to be set.
