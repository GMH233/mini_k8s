# 使用官方的 Golang 镜像作为基础镜像
FROM golang:bullseye AS builder

# 设置工作目录
WORKDIR /app

# 复制源代码到工作目录
COPY . .

WORKDIR /app/cmd/envoyinit
# 编译 Go 程序
RUN go build -o /envoy-init

# 使用官方的 Debian 镜像作为基础镜像
FROM debian:bullseye

USER root

# 安装 iptables
RUN apt-get update && apt-get install -y iptables && rm -rf /var/lib/apt/lists/*

# 设置工作目录
WORKDIR /app

# 从构建阶段复制编译后的 Go 程序
COPY --from=builder /envoy-init .

## 创建用户和组，并设置 UID 为 1337
#RUN groupadd -g 1337 envoy_proxy_group && useradd -u 1337 -g envoy_proxy_group -m envoy_proxy
#
## 更改文件所有者为新创建的用户和组
#RUN chown -R envoy_proxy:envoy_proxy_group /app
#
## 切换到非特权用户
#USER envoy_proxy

# 设置默认命令
CMD ["/app/envoy-init"]
