#!/bin/bash
set -e

echo "======================================"
echo "  VoiceChat 生产环境部署脚本"
echo "======================================"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}请使用 sudo 运行此脚本${NC}"
    exit 1
fi

# 检测操作系统
if [ -f /etc/debian_version ]; then
    OS="debian"
elif [ -f /etc/redhat-release ]; then
    OS="rhel"
else
    echo -e "${YELLOW}未检测到支持的操作系统${NC}"
    exit 1
fi

echo -e "${GREEN}检测到操作系统: $OS${NC}"

# 安装 Docker
echo -e "${GREEN}安装 Docker...${NC}"
if [ "$OS" == "debian" ]; then
    apt-get update
    apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release

    curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

    echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
elif [ "$OS" == "rhel" ]; then
    yum install -y yum-utils
    yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
    yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
fi

# 启动 Docker
echo -e "${GREEN}启动 Docker 服务...${NC}"
systemctl start docker
systemctl enable docker

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Docker 启动失败${NC}"
    exit 1
fi

echo -e "${GREEN}Docker 安装成功${NC}"

# 获取服务器 IP
SERVER_IP=$(hostname -I | awk '{print $1}')
echo -e "${GREEN}服务器 IP: $SERVER_IP${NC}"

# 创建部署目录
DEPLOY_DIR="/opt/fireredchat"
echo -e "${GREEN}创建部署目录: $DEPLOY_DIR${NC}"
mkdir -p $DEPLOY_DIR

# 复制项目文件
echo -e "${GREEN}复制项目文件...${NC}"
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cp -r $SCRIPT_DIR/.. $DEPLOY_DIR/

# 配置环境变量
echo -e "${GREEN}配置环境变量...${NC}"
cp $DEPLOY_DIR/deploy/.env.example $DEPLOY_DIR/deploy/.env

# 生成随机密码
JWT_SECRET=$(openssl rand -base64 32)
DB_PASSWORD=$(openssl rand -base64 16)
REDIS_PASSWORD=$(openssl rand -base64 16)

sed -i "s/DB_PASSWORD=.*/DB_PASSWORD=$DB_PASSWORD/" $DEPLOY_DIR/deploy/.env
sed -i "s/REDIS_PASSWORD=.*/REDIS_PASSWORD=$REDIS_PASSWORD/" $DEPLOY_DIR/deploy/.ploy
sed -i "s/JWT_SECRET=.*/JWT_SECRET=$JWT_SECRET/" $DEPLOY_DIR/deploy/.env

# 开放端口
echo -e "${GREEN}配置防火墙...${NC}"
if command -v ufw &> /dev/null; then
    ufw allow 80/tcp
    ufw allow 443/tcp
    ufw allow 3000/tcp
    ufw allow 8080/tcp
    ufw allow 8081/tcp
    ufw allow 5432/tcp
    ufw allow 6379/tcp
elif command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --add-port=80/tcp
    firewall-cmd --permanent --add-port=443/tcp
    firewall-cmd --permanent --add-port=3000/tcp
    firewall-cmd --permanent --add-port=8080/tcp
    firewall-cmd --permanent --add-port=8081/tcp
    firewall-cmd --permanent --add-port=5432/tcp
    firewall-cmd --permanent --add-port=6379/tcp
    firewall-cmd --reload
fi

# 启动服务
echo -e "${GREEN}启动服务...${NC}"
cd $DEPLOY_DIR/deploy
docker-compose -f docker-compose.prod.yaml up -d --build

# 等待服务启动
echo -e "${GREEN}等待服务启动...${NC}"
sleep 10

# 检查服务状态
echo -e "${GREEN}检查服务状态...${NC}"
docker-compose -f docker-compose.prod.yaml ps

# 完成
echo ""
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}  部署完成！${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""
echo -e "前端地址: http://$SERVER_IP:3000"
echo -e "API 地址: http://$SERVER_IP:8080/api/v1"
echo -e "WebSocket: ws://$SERVER_IP:8081"
echo ""
echo -e "配置文件位置: $DEPLOY_DIR/deploy/.env"
echo -e "查看日志: cd $DEPLOY_DIR/deploy && docker-compose -f docker-compose.prod.yaml logs -f"
echo ""
