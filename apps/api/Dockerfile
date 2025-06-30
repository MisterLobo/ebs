FROM golang:latest as base

WORKDIR /usr/src/workdir
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -race

FROM base AS final
WORKDIR /dist
COPY --from=base /usr/src/workdir/dist .

CMD ["./api"]