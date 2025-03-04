FROM debian:stable-slim

COPY "pomoteam" "/bin/pomoteam"

CMD ["/bin/pomoteam"]

