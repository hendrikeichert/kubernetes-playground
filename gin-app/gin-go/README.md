# build
go build -o gin-go ./cmd

# test
curl http://localhost:8080/users

# build docker image
docker build -t 192.168.1.100:32768/go-gin:v250731 .

# Docker registry
http://192.168.1.100:32768/v2/_catalog