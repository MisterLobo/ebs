FROM golang:latest as base

EXPOSE 9090
EXPOSE 443

WORKDIR /workdir
COPY apps/api/ .

RUN ls -al

RUN go mod download
RUN go build -o dist/api src/*.go

FROM base AS final
ENV PORT=9090
WORKDIR /dist
# COPY --from=base /workdir/admin-sdk-credentials.json .
COPY --from=base /workdir/dist .

CMD ["./api"]