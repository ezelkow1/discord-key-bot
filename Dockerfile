from golang:1.16-alpine

WORKDIR /app
COPY go.mod ./
COPY *.go ./
COPY *.sum ./

RUN go mod download
RUN go build -o /discord-key-bot
CMD /discord-key-bot -c /config/conf.json
