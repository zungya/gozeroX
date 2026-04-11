#!/bin/bash
# gozeroX 构建所有服务镜像
# 使用方法: bash build-images.sh
# 前提: 已安装 Docker

set -e

REGISTRY="${REGISTRY:-gozerox}"
TAG="${TAG:-latest}"

build() {
    local name=$1
    local service_path=$2
    echo "构建 ${REGISTRY}/${name}:${TAG} ..."
    docker build --build-arg BUILD_SERVICE=${service_path} -t "${REGISTRY}/${name}:${TAG}" .
    echo "✓ ${REGISTRY}/${name}:${TAG} 构建完成"
    echo ""
}

echo "=========================================="
echo "构建 gozeroX 所有服务镜像"
echo "=========================================="
echo ""

# API 服务
build usercenter-api        ./app/usercenter/cmd/api
build contentservice-api    ./app/contentService/cmd/api
build interactservice-api   ./app/interactService/cmd/api
build noticeservice-api     ./app/noticeService/cmd/api
build recommendservice-api  ./app/recommendService/cmd/api

# RPC 服务
build usercenter-rpc        ./app/usercenter/cmd/rpc
build contentservice-rpc    ./app/contentService/cmd/rpc
build interactservice-rpc   ./app/interactService/cmd/rpc
build noticeservice-rpc     ./app/noticeService/cmd/rpc
build recommendservice-rpc  ./app/recommendService/cmd/rpc

# MQ 消费者
build interactservice-mq    ./app/interactService/cmd/mq
build noticeservice-mq      ./app/noticeService/cmd/mq

echo "=========================================="
echo "全部构建完成！"
echo ""
echo "镜像列表:"
docker images | grep "^gozerox" | awk '{printf "  %-40s %s\n", $1, $2}'
echo ""
echo "Python 召回服务请到 PyRecommend/ 目录单独构建"
