# To enable ssh & remote debugging on app service change the base image to the one below
FROM ubuntu:18.04
COPY /bin/cnabtoarmtemplate /
WORKDIR /
RUN apt update && apt upgrade -y && apt install curl -y
ENTRYPOINT ["/cnabtoarmtemplate","listen"]