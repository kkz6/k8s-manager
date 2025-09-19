FROM alpine:3.22
COPY golang-cli-template /usr/bin/golang-cli-template
ENTRYPOINT ["/usr/bin/golang-cli-template"]