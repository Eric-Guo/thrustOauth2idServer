FROM alpine:latest
MAINTAINER zhufuyi "g.zhufuyi@gmail.com"

# set the time zone to Shanghai
RUN apk add tzdata  \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

# add curl, used for http service checking, can be installed without it if deployed in k8s
RUN apk add curl

COPY configs/ /app/configs/
COPY thrustOauth2idServer /app/thrustOauth2idServer
RUN chmod +x /app/thrustOauth2idServer

# http port
EXPOSE 8080


WORKDIR /app

CMD ["./thrustOauth2idServer", "-c", "configs/thrustOauth2idServer.yml"]
# if you use the Configuration Center, thrustOauth2idServer.yml is changed to the Configuration Center configuration.
#CMD ["./thrustOauth2idServer", "-c", "configs/thrustOauth2idServer.yml", "-enable-cc"]
