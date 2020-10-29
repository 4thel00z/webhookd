FROM golang 

ENV GO111MODULE=on

WORKDIR /app

COPY . .

RUN scripts/before_docker.sh
