FROM golang:1.24
WORKDIR /app
COPY . .
RUN go build -o gassigeher ./cmd/server
EXPOSE 8888
CMD ./gassigeher