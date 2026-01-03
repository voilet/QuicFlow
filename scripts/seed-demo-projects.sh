#!/bin/bash

# Demo Projects Seed Script
# 创建四种部署类型的演示项目和版本

set -e

API_BASE="${API_BASE:-http://localhost:8475/api/release}"

echo "=== 创建发布管理演示项目和版本 ==="
echo "API: $API_BASE"
echo ""

# Helper function to create project and get ID
create_project() {
    local data="$1"
    local response=$(curl -s -X POST "$API_BASE/projects" \
        -H "Content-Type: application/json" \
        -d "$data")
    echo "$response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4
}

# Helper function to create version
create_version() {
    local project_id="$1"
    local data="$2"
    curl -s -X POST "$API_BASE/projects/$project_id/versions" \
        -H "Content-Type: application/json" \
        -d "$data"
}

# ==================== 1. 脚本部署项目 ====================
echo ">>> 创建脚本部署项目..."

SCRIPT_PROJECT=$(cat <<'EOF'
{
    "name": "Web Application (Script)",
    "description": "基于脚本的 Web 应用部署示例，支持安装、更新、回滚和卸载操作",
    "type": "script",
    "repo_url": "https://github.com/example/webapp.git",
    "repo_type": "git",
    "script_config": {
        "work_dir": "/opt/webapp",
        "interpreter": "bash",
        "environment": {
            "APP_ENV": "production",
            "LOG_LEVEL": "info"
        },
        "timeouts": {
            "install": 300,
            "update": 180,
            "rollback": 120,
            "uninstall": 60
        }
    }
}
EOF
)

SCRIPT_PROJECT_ID=$(create_project "$SCRIPT_PROJECT")
echo "   项目ID: $SCRIPT_PROJECT_ID"

# 脚本部署 - 版本 1.0.0
SCRIPT_VERSION_1=$(cat <<'EOF'
{
    "version": "1.0.0",
    "description": "初始发布版本 - 基础功能",
    "work_dir": "/opt/webapp",
    "install_script": "#!/bin/bash\nset -e\n\necho \"[安装] Web Application v1.0.0\"\necho \"工作目录: $PWD\"\n\n# 创建目录结构\nmkdir -p logs config data\n\n# 下载应用包\necho \"下载应用包...\"\n# curl -L https://releases.example.com/webapp-1.0.0.tar.gz -o app.tar.gz\n# tar -xzf app.tar.gz\n\n# 配置应用\ncat > config/app.conf << 'CONF'\nserver.port=8080\nserver.host=0.0.0.0\nlog.level=info\nCONF\n\n# 创建启动脚本\ncat > start.sh << 'START'\n#!/bin/bash\ncd /opt/webapp\nnohup ./bin/webapp > logs/app.log 2>&1 &\necho $! > app.pid\nSTART\nchmod +x start.sh\n\necho \"[安装完成]\"",
    "update_script": "#!/bin/bash\nset -e\n\necho \"[更新] Web Application 到 v1.0.0\"\n\n# 备份当前版本\nif [ -d \"bin\" ]; then\n    cp -r bin bin.bak\nfi\n\n# 停止服务\nif [ -f \"app.pid\" ]; then\n    kill $(cat app.pid) 2>/dev/null || true\n    rm -f app.pid\nfi\n\n# 下载新版本\necho \"下载新版本...\"\n# curl -L https://releases.example.com/webapp-1.0.0.tar.gz -o app.tar.gz\n# tar -xzf app.tar.gz\n\n# 启动服务\n./start.sh\n\necho \"[更新完成]\"",
    "rollback_script": "#!/bin/bash\nset -e\n\necho \"[回滚] 恢复上一版本\"\n\n# 停止当前版本\nif [ -f \"app.pid\" ]; then\n    kill $(cat app.pid) 2>/dev/null || true\nfi\n\n# 恢复备份\nif [ -d \"bin.bak\" ]; then\n    rm -rf bin\n    mv bin.bak bin\n    echo \"已恢复备份版本\"\nelse\n    echo \"错误: 没有找到备份\"\n    exit 1\nfi\n\n# 启动服务\n./start.sh\n\necho \"[回滚完成]\"",
    "uninstall_script": "#!/bin/bash\nset -e\n\necho \"[卸载] Web Application\"\n\n# 停止服务\nif [ -f \"app.pid\" ]; then\n    kill $(cat app.pid) 2>/dev/null || true\nfi\n\n# 清理文件\nrm -rf bin logs data config app.pid start.sh\n\necho \"[卸载完成]\"",
    "skip_validation": true
}
EOF
)

echo "   创建版本 1.0.0..."
create_version "$SCRIPT_PROJECT_ID" "$SCRIPT_VERSION_1" > /dev/null

# 脚本部署 - 版本 1.1.0
SCRIPT_VERSION_2=$(cat <<'EOF'
{
    "version": "1.1.0",
    "description": "功能更新 - 新增缓存支持和性能优化",
    "work_dir": "/opt/webapp",
    "install_script": "#!/bin/bash\nset -e\n\necho \"[安装] Web Application v1.1.0\"\necho \"工作目录: $PWD\"\n\n# 创建目录结构\nmkdir -p logs config data cache\n\n# 配置应用\ncat > config/app.conf << 'CONF'\nserver.port=8080\nserver.host=0.0.0.0\nlog.level=info\ncache.enabled=true\ncache.ttl=3600\nCONF\n\n# 创建启动脚本\ncat > start.sh << 'START'\n#!/bin/bash\ncd /opt/webapp\nexport CACHE_ENABLED=true\nnohup ./bin/webapp > logs/app.log 2>&1 &\necho $! > app.pid\nSTART\nchmod +x start.sh\n\necho \"[安装完成]\"",
    "update_script": "#!/bin/bash\nset -e\n\necho \"[更新] Web Application 到 v1.1.0\"\n\n# 备份\ncp -r bin bin.bak 2>/dev/null || true\ncp config/app.conf config/app.conf.bak 2>/dev/null || true\n\n# 停止服务\nif [ -f \"app.pid\" ]; then\n    kill $(cat app.pid) 2>/dev/null || true\nfi\n\n# 创建缓存目录\nmkdir -p cache\n\n# 更新配置\nif ! grep -q \"cache.enabled\" config/app.conf; then\n    echo \"cache.enabled=true\" >> config/app.conf\n    echo \"cache.ttl=3600\" >> config/app.conf\nfi\n\n# 启动服务\n./start.sh\n\necho \"[更新完成]\"",
    "rollback_script": "#!/bin/bash\nset -e\n\necho \"[回滚] 恢复到 v1.0.0\"\n\nif [ -f \"app.pid\" ]; then\n    kill $(cat app.pid) 2>/dev/null || true\nfi\n\nif [ -d \"bin.bak\" ]; then\n    rm -rf bin && mv bin.bak bin\nfi\nif [ -f \"config/app.conf.bak\" ]; then\n    mv config/app.conf.bak config/app.conf\nfi\n\n./start.sh\necho \"[回滚完成]\"",
    "uninstall_script": "#!/bin/bash\nset -e\n\necho \"[卸载] Web Application v1.1.0\"\n\nif [ -f \"app.pid\" ]; then\n    kill $(cat app.pid) 2>/dev/null || true\nfi\n\nrm -rf bin logs data config cache app.pid start.sh\n\necho \"[卸载完成]\"",
    "skip_validation": true
}
EOF
)

echo "   创建版本 1.1.0..."
create_version "$SCRIPT_PROJECT_ID" "$SCRIPT_VERSION_2" > /dev/null

# 脚本部署 - 版本 2.0.0
SCRIPT_VERSION_3=$(cat <<'EOF'
{
    "version": "2.0.0",
    "description": "大版本更新 - 架构重构，支持集群部署",
    "work_dir": "/opt/webapp",
    "install_script": "#!/bin/bash\nset -e\n\necho \"[安装] Web Application v2.0.0 (集群版)\"\n\n# 创建目录\nmkdir -p logs config data cache cluster\n\n# 集群配置\ncat > config/cluster.conf << 'CONF'\ncluster.enabled=true\ncluster.node_id=${HOSTNAME}\ncluster.discovery=consul://consul:8500\nCONF\n\n# 应用配置\ncat > config/app.conf << 'CONF'\nserver.port=8080\nlog.level=info\ncache.enabled=true\ncache.type=redis\ncluster.config=config/cluster.conf\nCONF\n\necho \"[安装完成]\"",
    "update_script": "#!/bin/bash\nset -e\n\necho \"[更新] Web Application 到 v2.0.0\"\n\n# 备份\ntar -czf backup-$(date +%Y%m%d%H%M%S).tar.gz bin config 2>/dev/null || true\n\n# 停止服务\nif [ -f \"app.pid\" ]; then\n    kill $(cat app.pid) 2>/dev/null || true\nfi\n\n# 创建新目录\nmkdir -p cluster\n\n# 更新配置（迁移到集群模式）\nif [ ! -f \"config/cluster.conf\" ]; then\n    cat > config/cluster.conf << 'CONF'\ncluster.enabled=true\ncluster.node_id=${HOSTNAME}\nCONF\nfi\n\n./start.sh\necho \"[更新完成]\"",
    "rollback_script": "#!/bin/bash\nset -e\n\necho \"[回滚] 恢复之前版本\"\n\n# 查找最新备份\nBACKUP=$(ls -t backup-*.tar.gz 2>/dev/null | head -1)\nif [ -z \"$BACKUP\" ]; then\n    echo \"错误: 没有找到备份文件\"\n    exit 1\nfi\n\n# 停止服务\nif [ -f \"app.pid\" ]; then\n    kill $(cat app.pid) 2>/dev/null || true\nfi\n\n# 恢复备份\ntar -xzf \"$BACKUP\"\n\n./start.sh\necho \"[回滚完成] 从 $BACKUP 恢复\"",
    "uninstall_script": "#!/bin/bash\nset -e\n\necho \"[卸载] Web Application v2.0.0\"\n\nif [ -f \"app.pid\" ]; then\n    kill $(cat app.pid) 2>/dev/null || true\nfi\n\nrm -rf bin logs data config cache cluster app.pid start.sh backup-*.tar.gz\n\necho \"[卸载完成]\"",
    "skip_validation": true
}
EOF
)

echo "   创建版本 2.0.0..."
create_version "$SCRIPT_PROJECT_ID" "$SCRIPT_VERSION_3" > /dev/null

# 脚本部署 - 版本 2.1.0
SCRIPT_VERSION_4=$(cat <<'EOF'
{
    "version": "2.1.0",
    "description": "安全更新 - 修复安全漏洞，增强日志审计",
    "work_dir": "/opt/webapp",
    "install_script": "#!/bin/bash\nset -e\n\necho \"[安装] Web Application v2.1.0 (安全增强版)\"\n\nmkdir -p logs config data cache cluster audit\n\n# 审计配置\ncat > config/audit.conf << 'CONF'\naudit.enabled=true\naudit.log_path=audit/access.log\naudit.retention_days=90\nCONF\n\necho \"[安装完成]\"",
    "update_script": "#!/bin/bash\nset -e\n\necho \"[更新] Web Application 到 v2.1.0 (安全补丁)\"\n\n# 备份\ntar -czf backup-$(date +%Y%m%d%H%M%S).tar.gz bin config\n\n# 停止\nif [ -f \"app.pid\" ]; then kill $(cat app.pid) 2>/dev/null || true; fi\n\n# 创建审计目录\nmkdir -p audit\n\n# 添加审计配置\nif [ ! -f \"config/audit.conf\" ]; then\n    cat > config/audit.conf << 'CONF'\naudit.enabled=true\naudit.log_path=audit/access.log\nCONF\nfi\n\n./start.sh\necho \"[更新完成] 安全补丁已应用\"",
    "rollback_script": "#!/bin/bash\nset -e\n\nBACKUP=$(ls -t backup-*.tar.gz 2>/dev/null | head -1)\nif [ -z \"$BACKUP\" ]; then exit 1; fi\nif [ -f \"app.pid\" ]; then kill $(cat app.pid) 2>/dev/null || true; fi\ntar -xzf \"$BACKUP\"\n./start.sh\necho \"[回滚完成]\"",
    "uninstall_script": "#!/bin/bash\nset -e\nif [ -f \"app.pid\" ]; then kill $(cat app.pid) 2>/dev/null || true; fi\nrm -rf bin logs data config cache cluster audit app.pid start.sh backup-*.tar.gz\necho \"[卸载完成]\"",
    "skip_validation": true
}
EOF
)

echo "   创建版本 2.1.0..."
create_version "$SCRIPT_PROJECT_ID" "$SCRIPT_VERSION_4" > /dev/null
echo "   脚本部署项目创建完成"
echo ""

# ==================== 2. 容器部署项目 ====================
echo ">>> 创建容器部署项目..."

CONTAINER_PROJECT=$(cat <<'EOF'
{
    "name": "API Gateway (Docker)",
    "description": "基于 Docker 容器的 API 网关服务，支持自动拉取镜像和健康检查",
    "type": "container",
    "container_config": {
        "image": "nginx:alpine",
        "registry": "docker.io",
        "image_pull_policy": "always",
        "container_name": "api-gateway",
        "environment": {
            "TZ": "Asia/Shanghai",
            "NGINX_WORKER_PROCESSES": "auto"
        },
        "ports": [
            {"host_port": 80, "container_port": 80, "protocol": "tcp"},
            {"host_port": 443, "container_port": 443, "protocol": "tcp"}
        ],
        "volumes": [
            {"host_path": "/data/nginx/conf", "container_path": "/etc/nginx/conf.d", "read_only": true},
            {"host_path": "/data/nginx/logs", "container_path": "/var/log/nginx", "read_only": false}
        ],
        "memory_limit": "512m",
        "cpu_limit": "1",
        "restart_policy": "unless-stopped",
        "health_check": {
            "command": ["CMD", "curl", "-f", "http://localhost/health"],
            "interval": 30,
            "timeout": 10,
            "retries": 3,
            "start_period": 10
        },
        "labels": {
            "app": "api-gateway",
            "env": "production"
        },
        "networks": ["app-network"],
        "log_driver": "json-file",
        "log_opts": {
            "max-size": "100m",
            "max-file": "3"
        }
    }
}
EOF
)

CONTAINER_PROJECT_ID=$(create_project "$CONTAINER_PROJECT")
echo "   项目ID: $CONTAINER_PROJECT_ID"

# 容器部署 - 版本 1.0.0
CONTAINER_VERSION_1=$(cat <<'EOF'
{
    "version": "1.0.0",
    "description": "初始版本 - Nginx 1.24 稳定版",
    "container_image": "nginx:1.24-alpine",
    "deploy_config": {
        "image": "nginx:1.24-alpine",
        "environment": {
            "NGINX_VERSION": "1.24"
        },
        "resources": {
            "cpu_limit": "500m",
            "memory_limit": "256Mi"
        },
        "health_check": {
            "command": ["CMD", "wget", "-q", "--spider", "http://localhost/"],
            "interval": 30,
            "timeout": 5,
            "retries": 3
        }
    }
}
EOF
)

echo "   创建版本 1.0.0..."
create_version "$CONTAINER_PROJECT_ID" "$CONTAINER_VERSION_1" > /dev/null

# 容器部署 - 版本 1.1.0
CONTAINER_VERSION_2=$(cat <<'EOF'
{
    "version": "1.1.0",
    "description": "性能优化 - 增加 Worker 数量和缓存配置",
    "container_image": "nginx:1.24-alpine",
    "deploy_config": {
        "image": "nginx:1.24-alpine",
        "environment": {
            "NGINX_VERSION": "1.24",
            "NGINX_WORKER_PROCESSES": "4",
            "PROXY_CACHE_PATH": "/var/cache/nginx"
        },
        "resources": {
            "cpu_limit": "1",
            "memory_limit": "512Mi"
        }
    }
}
EOF
)

echo "   创建版本 1.1.0..."
create_version "$CONTAINER_PROJECT_ID" "$CONTAINER_VERSION_2" > /dev/null

# 容器部署 - 版本 1.25.0
CONTAINER_VERSION_3=$(cat <<'EOF'
{
    "version": "1.25.0",
    "description": "升级到 Nginx 1.25 - 支持 HTTP/3",
    "container_image": "nginx:1.25-alpine",
    "deploy_config": {
        "image": "nginx:1.25-alpine",
        "environment": {
            "NGINX_VERSION": "1.25",
            "ENABLE_HTTP3": "true"
        },
        "resources": {
            "cpu_limit": "1",
            "memory_limit": "512Mi"
        },
        "health_check": {
            "command": ["CMD", "nginx", "-t"],
            "interval": 30,
            "timeout": 10,
            "retries": 3
        }
    }
}
EOF
)

echo "   创建版本 1.25.0..."
create_version "$CONTAINER_PROJECT_ID" "$CONTAINER_VERSION_3" > /dev/null

# 容器部署 - 版本 1.26.0
CONTAINER_VERSION_4=$(cat <<'EOF'
{
    "version": "1.26.0",
    "description": "最新稳定版 - 安全增强和性能改进",
    "container_image": "nginx:1.26-alpine",
    "deploy_config": {
        "image": "nginx:1.26-alpine",
        "environment": {
            "NGINX_VERSION": "1.26",
            "ENABLE_HTTP3": "true",
            "SECURITY_HEADERS": "strict"
        },
        "resources": {
            "cpu_limit": "2",
            "memory_limit": "1Gi"
        }
    }
}
EOF
)

echo "   创建版本 1.26.0..."
create_version "$CONTAINER_PROJECT_ID" "$CONTAINER_VERSION_4" > /dev/null
echo "   容器部署项目创建完成"
echo ""

# ==================== 3. Git 拉取部署项目 ====================
echo ">>> 创建 Git 拉取部署项目..."

GITPULL_PROJECT=$(cat <<'EOF'
{
    "name": "Frontend App (Git Pull)",
    "description": "前端应用 Git 拉取部署，支持自动构建和部署",
    "type": "gitpull",
    "repo_url": "https://github.com/example/frontend-app.git",
    "repo_type": "git",
    "gitpull_config": {
        "repo_url": "https://github.com/example/frontend-app.git",
        "branch": "main",
        "depth": 1,
        "submodules": false,
        "auth_type": "token",
        "work_dir": "/var/www/frontend",
        "clean_before": false,
        "backup_before": true,
        "backup_dir": "/var/www/backups",
        "backup_keep": 5,
        "pre_script": "#!/bin/bash\necho \"准备部署...\"\nnpm --version || echo \"npm not found\"",
        "post_script": "#!/bin/bash\necho \"开始构建...\"\nif [ -f \"package.json\" ]; then\n    npm install\n    npm run build\nfi\necho \"部署完成\"",
        "environment": {
            "NODE_ENV": "production",
            "CI": "true"
        },
        "interpreter": "bash",
        "clone_timeout": 120,
        "script_timeout": 300
    }
}
EOF
)

GITPULL_PROJECT_ID=$(create_project "$GITPULL_PROJECT")
echo "   项目ID: $GITPULL_PROJECT_ID"

# Git 拉取 - 版本基于 tag v1.0.0
GITPULL_VERSION_1=$(cat <<'EOF'
{
    "version": "v1.0.0",
    "description": "初始发布 - React 18 基础版本",
    "git_ref": "v1.0.0",
    "git_ref_type": "tag",
    "work_dir": "/var/www/frontend",
    "install_script": "#!/bin/bash\nset -e\necho \"部署前端应用 v1.0.0\"\nnpm ci --production\nnpm run build\ncp -r dist/* /var/www/html/\necho \"部署完成\"",
    "skip_validation": true
}
EOF
)

echo "   创建版本 v1.0.0..."
create_version "$GITPULL_PROJECT_ID" "$GITPULL_VERSION_1" > /dev/null

# Git 拉取 - 版本基于 tag v1.1.0
GITPULL_VERSION_2=$(cat <<'EOF'
{
    "version": "v1.1.0",
    "description": "功能更新 - 新增用户中心模块",
    "git_ref": "v1.1.0",
    "git_ref_type": "tag",
    "work_dir": "/var/www/frontend",
    "install_script": "#!/bin/bash\nset -e\necho \"部署前端应用 v1.1.0\"\nexport NODE_OPTIONS='--max_old_space_size=4096'\nnpm ci\nnpm run build:prod\ncp -r dist/* /var/www/html/\necho \"清理缓存...\"\nrm -rf /var/www/html/.cache\necho \"部署完成\"",
    "skip_validation": true
}
EOF
)

echo "   创建版本 v1.1.0..."
create_version "$GITPULL_PROJECT_ID" "$GITPULL_VERSION_2" > /dev/null

# Git 拉取 - 版本基于分支 develop
GITPULL_VERSION_3=$(cat <<'EOF'
{
    "version": "v2.0.0-beta",
    "description": "Beta 版本 - 新 UI 框架迁移 (从 develop 分支)",
    "git_ref": "develop",
    "git_ref_type": "branch",
    "work_dir": "/var/www/frontend-beta",
    "install_script": "#!/bin/bash\nset -e\necho \"部署 Beta 版本\"\nexport NODE_ENV=staging\nnpm ci\nnpm run build:staging\nmkdir -p /var/www/html-beta\ncp -r dist/* /var/www/html-beta/\necho \"Beta 版本部署完成\"",
    "skip_validation": true
}
EOF
)

echo "   创建版本 v2.0.0-beta..."
create_version "$GITPULL_PROJECT_ID" "$GITPULL_VERSION_3" > /dev/null

# Git 拉取 - 版本基于 tag v2.0.0
GITPULL_VERSION_4=$(cat <<'EOF'
{
    "version": "v2.0.0",
    "description": "正式发布 - 全新 UI 框架，性能大幅提升",
    "git_ref": "v2.0.0",
    "git_ref_type": "tag",
    "work_dir": "/var/www/frontend",
    "install_script": "#!/bin/bash\nset -e\necho \"部署前端应用 v2.0.0\"\n\n# 安装依赖\nnpm ci --production\n\n# 构建\nexport NODE_OPTIONS='--max_old_space_size=4096'\nnpm run build:prod\n\n# 部署\nBACKUP_DIR=\"/var/www/html.bak.$(date +%Y%m%d%H%M%S)\"\nmv /var/www/html \"$BACKUP_DIR\" 2>/dev/null || true\nmkdir -p /var/www/html\ncp -r dist/* /var/www/html/\n\n# 清理旧备份 (保留最近5个)\nls -dt /var/www/html.bak.* 2>/dev/null | tail -n +6 | xargs rm -rf 2>/dev/null || true\n\necho \"部署完成\"",
    "skip_validation": true
}
EOF
)

echo "   创建版本 v2.0.0..."
create_version "$GITPULL_PROJECT_ID" "$GITPULL_VERSION_4" > /dev/null
echo "   Git 拉取部署项目创建完成"
echo ""

# ==================== 4. Kubernetes 部署项目 ====================
echo ">>> 创建 Kubernetes 部署项目..."

K8S_PROJECT=$(cat <<'EOF'
{
    "name": "Microservice API (Kubernetes)",
    "description": "微服务 API 的 Kubernetes 部署，支持自动扩缩容和滚动更新",
    "type": "kubernetes",
    "kubernetes_config": {
        "namespace": "production",
        "resource_type": "deployment",
        "resource_name": "microservice-api",
        "container_name": "api",
        "image": "myregistry.io/microservice-api:latest",
        "registry": "myregistry.io",
        "image_pull_policy": "Always",
        "image_pull_secret": "regcred",
        "replicas": 3,
        "update_strategy": "RollingUpdate",
        "max_unavailable": "25%",
        "max_surge": "25%",
        "min_ready_seconds": 10,
        "cpu_request": "100m",
        "cpu_limit": "500m",
        "memory_request": "128Mi",
        "memory_limit": "512Mi",
        "environment": {
            "APP_ENV": "production",
            "LOG_LEVEL": "info",
            "DB_HOST": "postgres-service"
        },
        "service_type": "ClusterIP",
        "service_ports": [
            {"name": "http", "port": 80, "target_port": 8080, "protocol": "TCP"},
            {"name": "grpc", "port": 9090, "target_port": 9090, "protocol": "TCP"}
        ],
        "deploy_timeout": 300,
        "rollout_timeout": 600
    }
}
EOF
)

K8S_PROJECT_ID=$(create_project "$K8S_PROJECT")
echo "   项目ID: $K8S_PROJECT_ID"

# K8s 部署 - 版本 1.0.0
K8S_VERSION_1=$(cat <<'EOF'
{
    "version": "1.0.0",
    "description": "初始发布 - 基础 API 服务",
    "container_image": "myregistry.io/microservice-api:1.0.0",
    "replicas": 2,
    "k8s_yaml": "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: microservice-api\n  namespace: production\n  labels:\n    app: microservice-api\n    version: v1.0.0\nspec:\n  replicas: 2\n  selector:\n    matchLabels:\n      app: microservice-api\n  template:\n    metadata:\n      labels:\n        app: microservice-api\n        version: v1.0.0\n    spec:\n      containers:\n      - name: api\n        image: myregistry.io/microservice-api:1.0.0\n        ports:\n        - containerPort: 8080\n          name: http\n        resources:\n          requests:\n            cpu: 100m\n            memory: 128Mi\n          limits:\n            cpu: 500m\n            memory: 512Mi\n        livenessProbe:\n          httpGet:\n            path: /health\n            port: 8080\n          initialDelaySeconds: 15\n          periodSeconds: 20\n        readinessProbe:\n          httpGet:\n            path: /ready\n            port: 8080\n          initialDelaySeconds: 5\n          periodSeconds: 10",
    "deploy_config": {
        "image": "myregistry.io/microservice-api:1.0.0",
        "replicas": 2,
        "environment": {
            "VERSION": "1.0.0"
        },
        "resources": {
            "cpu_request": "100m",
            "cpu_limit": "500m",
            "memory_request": "128Mi",
            "memory_limit": "512Mi"
        }
    }
}
EOF
)

echo "   创建版本 1.0.0..."
create_version "$K8S_PROJECT_ID" "$K8S_VERSION_1" > /dev/null

# K8s 部署 - 版本 1.1.0
K8S_VERSION_2=$(cat <<'EOF'
{
    "version": "1.1.0",
    "description": "功能更新 - 新增缓存层和消息队列集成",
    "container_image": "myregistry.io/microservice-api:1.1.0",
    "replicas": 3,
    "k8s_yaml": "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: microservice-api\n  namespace: production\nspec:\n  replicas: 3\n  selector:\n    matchLabels:\n      app: microservice-api\n  template:\n    metadata:\n      labels:\n        app: microservice-api\n        version: v1.1.0\n    spec:\n      containers:\n      - name: api\n        image: myregistry.io/microservice-api:1.1.0\n        ports:\n        - containerPort: 8080\n        env:\n        - name: REDIS_HOST\n          value: redis-service\n        - name: MQ_HOST\n          value: rabbitmq-service\n        resources:\n          requests:\n            cpu: 200m\n            memory: 256Mi\n          limits:\n            cpu: 1\n            memory: 1Gi",
    "deploy_config": {
        "image": "myregistry.io/microservice-api:1.1.0",
        "replicas": 3,
        "environment": {
            "VERSION": "1.1.0",
            "REDIS_ENABLED": "true",
            "MQ_ENABLED": "true"
        },
        "resources": {
            "cpu_request": "200m",
            "cpu_limit": "1",
            "memory_request": "256Mi",
            "memory_limit": "1Gi"
        }
    }
}
EOF
)

echo "   创建版本 1.1.0..."
create_version "$K8S_PROJECT_ID" "$K8S_VERSION_2" > /dev/null

# K8s 部署 - 版本 2.0.0
K8S_VERSION_3=$(cat <<'EOF'
{
    "version": "2.0.0",
    "description": "大版本更新 - gRPC 支持和自动扩缩容",
    "container_image": "myregistry.io/microservice-api:2.0.0",
    "replicas": 3,
    "k8s_yaml": "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: microservice-api\n  namespace: production\nspec:\n  replicas: 3\n  strategy:\n    type: RollingUpdate\n    rollingUpdate:\n      maxUnavailable: 1\n      maxSurge: 1\n  selector:\n    matchLabels:\n      app: microservice-api\n  template:\n    metadata:\n      labels:\n        app: microservice-api\n        version: v2.0.0\n    spec:\n      containers:\n      - name: api\n        image: myregistry.io/microservice-api:2.0.0\n        ports:\n        - containerPort: 8080\n          name: http\n        - containerPort: 9090\n          name: grpc\n        resources:\n          requests:\n            cpu: 250m\n            memory: 512Mi\n          limits:\n            cpu: 2\n            memory: 2Gi\n---\napiVersion: autoscaling/v2\nkind: HorizontalPodAutoscaler\nmetadata:\n  name: microservice-api-hpa\nspec:\n  scaleTargetRef:\n    apiVersion: apps/v1\n    kind: Deployment\n    name: microservice-api\n  minReplicas: 3\n  maxReplicas: 10\n  metrics:\n  - type: Resource\n    resource:\n      name: cpu\n      target:\n        type: Utilization\n        averageUtilization: 70",
    "deploy_config": {
        "image": "myregistry.io/microservice-api:2.0.0",
        "replicas": 3,
        "environment": {
            "VERSION": "2.0.0",
            "GRPC_ENABLED": "true",
            "HPA_ENABLED": "true"
        },
        "resources": {
            "cpu_request": "250m",
            "cpu_limit": "2",
            "memory_request": "512Mi",
            "memory_limit": "2Gi"
        }
    }
}
EOF
)

echo "   创建版本 2.0.0..."
create_version "$K8S_PROJECT_ID" "$K8S_VERSION_3" > /dev/null

# K8s 部署 - 版本 2.1.0
K8S_VERSION_4=$(cat <<'EOF'
{
    "version": "2.1.0",
    "description": "安全更新 - mTLS 和 Pod 安全策略",
    "container_image": "myregistry.io/microservice-api:2.1.0",
    "replicas": 3,
    "k8s_yaml": "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: microservice-api\n  namespace: production\n  annotations:\n    sidecar.istio.io/inject: \"true\"\nspec:\n  replicas: 3\n  selector:\n    matchLabels:\n      app: microservice-api\n  template:\n    metadata:\n      labels:\n        app: microservice-api\n        version: v2.1.0\n        security.istio.io/tlsMode: istio\n    spec:\n      securityContext:\n        runAsNonRoot: true\n        runAsUser: 1000\n        fsGroup: 1000\n      containers:\n      - name: api\n        image: myregistry.io/microservice-api:2.1.0\n        securityContext:\n          allowPrivilegeEscalation: false\n          readOnlyRootFilesystem: true\n          capabilities:\n            drop:\n            - ALL\n        ports:\n        - containerPort: 8080\n        - containerPort: 9090\n        volumeMounts:\n        - name: tmp\n          mountPath: /tmp\n        resources:\n          requests:\n            cpu: 250m\n            memory: 512Mi\n          limits:\n            cpu: 2\n            memory: 2Gi\n      volumes:\n      - name: tmp\n        emptyDir: {}",
    "deploy_config": {
        "image": "myregistry.io/microservice-api:2.1.0",
        "replicas": 3,
        "environment": {
            "VERSION": "2.1.0",
            "MTLS_ENABLED": "true",
            "SECURITY_MODE": "strict"
        },
        "resources": {
            "cpu_request": "250m",
            "cpu_limit": "2",
            "memory_request": "512Mi",
            "memory_limit": "2Gi"
        }
    }
}
EOF
)

echo "   创建版本 2.1.0..."
create_version "$K8S_PROJECT_ID" "$K8S_VERSION_4" > /dev/null
echo "   Kubernetes 部署项目创建完成"
echo ""

echo "=== 演示数据创建完成 ==="
echo ""
echo "已创建项目:"
echo "  1. Web Application (Script)        - 脚本部署示例 (4个版本)"
echo "  2. API Gateway (Docker)            - 容器部署示例 (4个版本)"
echo "  3. Frontend App (Git Pull)         - Git拉取部署示例 (4个版本)"
echo "  4. Microservice API (Kubernetes)   - K8s部署示例 (4个版本)"
echo ""
echo "项目 ID:"
echo "  Script:     $SCRIPT_PROJECT_ID"
echo "  Container:  $CONTAINER_PROJECT_ID"
echo "  Git Pull:   $GITPULL_PROJECT_ID"
echo "  Kubernetes: $K8S_PROJECT_ID"
