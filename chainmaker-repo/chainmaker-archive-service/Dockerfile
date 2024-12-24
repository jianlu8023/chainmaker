FROM golang:1.17.5 AS gobuilder

ENV GOPROXY=https://goproxy.cn,https://goproxy.io,direct
ENV GOPRIVATE=chainmaker.org/*
ENV GO111MODULE=on
ENV GONOSUMDB=chainmaker.org/*
ENV GOSUMDB=sum.golang.google.cn

WORKDIR /build/app

COPY go.* ./


RUN go mod download -x

COPY . .

RUN cd /build/app && \
    if [[ -f "chainmaker-archive-service" ]];then  rm app; fi && \
    go build -ldflags="-s -w" -o chainmaker-archive-service ./src

FROM ubuntu:20.04 AS datebuilder

RUN sed -i s@/archive.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list && \
     sed -i s@/security.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list && \
      apt update && \
      apt install tzdata -y && \
      echo "Asia/Shanghai" > /etc/timezone
RUN echo "done"


FROM ubuntu:20.04

WORKDIR /app

RUN sed -i s@/archive.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list && \
     sed -i s@/security.ubuntu.com/@/mirrors.aliyun.com/@g /etc/apt/sources.list && \
      apt update && \
      apt install curl -y

COPY --from=datebuilder /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
COPY --from=datebuilder /etc/timezone /etc/timezone
COPY --from=gobuilder /build/app/chainmaker-archive-service ./app
COPY --from=gobuilder /build/app/configs /app/configs

VOLUME /app/configs
VOLUME /app/log
VOLUME /app/service_datas

EXPOSE 13119
EXPOSE 13120
EXPOSE 13122
EXPOSE 13123

HEALTHCHECK --interval=60s --timeout=5s --retries=3 \
    CMD curl -X POST -ksS http://localhost:13119/health || exit 1

ENTRYPOINT ["./app"]
CMD ["-i", "./configs/config.yml"]
