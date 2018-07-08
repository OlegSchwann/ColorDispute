# take official image if go language
FROM golang:1.10
RUN ["go", "get", "github.com/gorilla/websocket"]
COPY server/ /server/
EXPOSE 8000
CMD ["go", "run", "/server/main.go", "/server/reactor.go"]
