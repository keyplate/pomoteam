FROM golang as build

WORKDIR /app

COPY . .

RUN go build


FROM debian:stable-slim

COPY --from=build "/app/pomoteam" "/bin/pomoteam"

CMD ["/bin/pomoteam"]

