FROM ubuntu:18.04
COPY /bin/cnabtoarmtemplate /
WORKDIR /
RUN apt update && apt upgrade -y && apt install curl -y
ENTRYPOINT ["/cnabtoarmtemplate","listen"]