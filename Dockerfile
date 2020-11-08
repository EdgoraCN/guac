 
FROM golang:alpine as builder

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk update && apk add curl git bash

WORKDIR /src

ENV GO111MODULE=on
ENV GOPROXY=https://mirrors.aliyun.com/goproxy/
#RUN git clone https://github.com/EdgoraCN/guac.git
COPY . /src/guac
RUN cd guac && go mod download && go build cmd/guac/guac.go



FROM itchyny/gojq as gojq

FROM edgora/guac-vue:1.3.2 as vue

FROM alpine:3

# Expose the application on port 4567*
EXPOSE 4567
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk update && apk add  --no-cache  curl bash

ENV GUACD guacd:4822
ENV CONFIG_PATH /app/config.yaml
ENV LOG_LEVEL INFO

COPY --from=builder /src/guac/config.yaml /app/config.yaml
COPY --from=builder /src/guac/guac /app/guac
COPY --from=vue /web /app/static

#COPY --from=builder /usr/local/bin/dockerize /usr/local/bin/dockerize
#COPY --from=gojq /usr/local/bin/gojq /usr/local/bin/gojq
WORKDIR /app

CMD ["/app/guac"]