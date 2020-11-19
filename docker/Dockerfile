FROM ubuntu:18.04

RUN apt-get update && \
    apt-get install -y curl iproute2 jq python3 && mkdir /application


RUN curl -sSL https://raw.githubusercontent.com/thombashi/tcconfig/master/scripts/installer.sh | bash


WORKDIR /application

# Adds node folder to path to access algorand binaries
ENV PATH="/application:${PATH}"

COPY algorand-go-implementation config.json /application/

ENTRYPOINT ["algorand-go-implementation"]
