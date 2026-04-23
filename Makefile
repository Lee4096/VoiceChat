.PHONY: help build build-linux build-darwin run stop clean test lint fmt check deps docker-build docker-up docker-down prod prod-down logs logs-fresh restart clean-all

# 项目名称和版本
BINARY_NAME=voicechat-server
VERSION?=latest
BUILD_TIME:=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_DIR:=./bin

# Go 配置
GO := go
GOFLAGS := -ldflags="-s -w -X 'main.Version=$(VERSION)' -X 'main.BuildTime=$(BUILD_TIME)'"

# Docker 配置
DOCKER_IMAGE := voicechat/$(BINARY_NAME)
DOCKER_TAG := $(VERSION)

# 默认目标
help:
	@echo "VoiceChat Server - 工业级部署系统"
	@echo ""
	@echo "可用命令:"
	@echo "  make build        - 构建 Linux amd64 二进制文件"
	@echo "  make run          - 直接运行服务"
	@echo "  make stop         - 停止运行的服务"
	@echo "  make clean        - 清理构建产物"
	@echo "  make test         - 运行测试"
	@echo "  make lint         - 运行代码检查"
	@echo "  make fmt          - 格式化代码"
	@echo "  make check        - 运行所有检查"
	@echo "  make deps         - 下载依赖"
	@echo "  make docker-build - 构建 Docker 镜像"
	@echo "  make docker-up    - 启动 Docker 容器"
	@echo "  make docker-down  - 停止 Docker 容器"
	@echo "  make prod         - 生产环境部署"
	@echo "  make prod-down    - 停止生产环境"
	@echo "  make logs         - 查看日志"
	@echo "  make restart      - 重启服务"
	@echo "  make clean-all    - 清理所有构建产物和 Docker"

# 构建
build:
	@echo "构建 Linux amd64 二进制文件..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

build-linux: build

build-darwin:
	@echo "构建 Darwin amd64 二进制文件..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/server

build-arm:
	@echo "构建 Linux ARM64 二进制文件..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-arm64 ./cmd/server

# 多架构构建
docker-build-multi:
	@echo "构建多架构 Docker 镜像..."
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		--push \
		.

# Docker 镜像扫描
docker-scan:
	@echo "扫描 Docker 镜像安全漏洞..."
	docker scout cves $(DOCKER_IMAGE):$(DOCKER_TAG)

# 运行
run:
	@echo "启动服务..."
	@mkdir -p logs
	./bin/$(BINARY_NAME)

stop:
	@echo "停止服务..."
	@pkill -f $(BINARY_NAME) || true
	@echo "服务已停止"

# 清理
clean:
	@echo "清理构建产物..."
	@rm -rf $(BUILD_DIR)
	@rm -rf logs/*.log
	@echo "清理完成"

# 测试
test:
	@echo "运行测试..."
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-coverage: test
	@echo "生成覆盖率报告..."
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 代码检查
lint:
	@echo "运行代码检查..."
	golangci-lint run ./...

fmt:
	@echo "格式化代码..."
	$(GO) fmt ./...

vet:
	@echo "运行 go vet..."
	$(GO) vet ./...

check: fmt vet lint test

# 依赖
deps:
	@echo "下载依赖..."
	$(GO) mod download
	$(GO) mod tidy

# Docker 构建
docker-build:
	@echo "构建 Docker 镜像: $(DOCKER_IMAGE):$(DOCKER_TAG)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker build -t $(DOCKER_IMAGE):latest .

docker-build-no-cache:
	@echo "无缓存构建 Docker 镜像..."
	docker build --no-cache -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

# Docker 运行
docker-up:
	@echo "启动 Docker 容器..."
	docker-compose up -d
	@echo "服务已启动"
	@docker-compose ps

docker-down:
	@echo "停止 Docker 容器..."
	docker-compose down

docker-restart: docker-down docker-up

# 生产环境部署
prod:
	@echo "部署生产环境..."
	@mkdir -p logs
	docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
	@echo "生产环境已部署"
	@docker-compose -f docker-compose.yml -f docker-compose.prod.yml ps

prod-down:
	@echo "停止生产环境..."
	docker-compose -f docker-compose.yml -f docker-compose.prod.yml down

prod-logs:
	docker-compose -f docker-compose.yml -f docker-compose.prod.yml logs -f

# 查看日志
logs:
	@if [ -f logs/voicechat.log ]; then tail -f logs/voicechat.log; else docker-compose logs -f; fi

logs-fresh:
	docker-compose logs -f --tail=100

# 重启
restart: stop run

# 完全清理
clean-all: clean
	@echo "清理 Docker..."
	docker-compose down -v --rmi local 2>/dev/null || true
	docker system prune -f
	@echo "清理完成"

# 打包发布
package:
	@echo "打包发布版本..."
	@mkdir -p release
	cd $(BUILD_DIR) && tar -czf ../release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)
	@echo "打包完成: release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz"

# 健康检查
health:
	@echo "检查服务健康状态..."
	@curl -s http://localhost:8080/health || echo "服务未响应"
	@curl -s http://localhost:8081/health || echo "WebSocket 服务未响应"

# 密钥生成
gen-secret:
	@echo "生成随机密钥..."
	@openssl rand -base64 32

# 数据库迁移 (预留)
migrate:
	@echo "运行数据库迁移..."
	# TODO: 实现迁移脚本

# 完整的开发环境设置
setup: deps build
	@echo "检查环境变量配置..."
	@if [ ! -f .env ]; then cp .env.example .env; echo "已创建 .env 文件，请编辑配置"; fi
	@echo "环境设置完成"
