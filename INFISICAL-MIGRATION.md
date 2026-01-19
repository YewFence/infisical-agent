# Infisical 迁移方案 - 全局 Agent 模式

## 背景

将 docker-config 目录下多个 compose 服务的敏感环境变量从 `.env` 文件迁移到 Infisical 统一管理。

### 选定方案
**全局 Infisical Agent 模式** - 部署一个独立的 infisical-agent 容器，所有服务从它生成的 secrets 文件读取。

---

## Infisical 项目结构（推荐）

在 Infisical 中按**文件夹**组织 secrets，每个服务一个文件夹：

```
Infisical 项目: "yewyard-server"
├── /vaultwarden/              # vaultwarden 专用
│   └── ADMIN_TOKEN=xxx
├── /nginx/                    # nginx 专用（示例）
│   └── SSL_PASSWORD=xxx
├── /postgres/                 # postgres 专用（示例）
│   └── POSTGRES_PASSWORD=xxx
└── ...
```

**好处**：
- 模板结构统一，添加新服务只需复制粘贴改路径
- 各服务 secrets 隔离清晰
- Infisical 界面管理方便

---

## 实施计划

### 第一步：在 Infisical 中配置

1. 登录 Infisical Cloud（https://app.infisical.com）
2. 创建项目（如 `yewyard-server`），记录**项目 ID**
3. 在项目中创建文件夹（如 `/vaultwarden`），添加 secrets
4. 创建 Machine Identity（机器身份）
   - 选择 Universal Auth 认证方式
   - 记录 `Client ID` 和 `Client Secret`
   - 授权该 Identity 访问项目

### 第二步：创建 Infisical Agent 服务

**目录结构** `infisical-agent/`

```
infisical-agent/
├── docker-compose.yml      # Agent 容器配置
├── config.yaml             # Agent 配置文件（由 generate 生成）
├── config.yaml.tmpl        # 配置模板
├── config.yaml           # 服务列表配置
├── generate.exe            # 配置生成器
├── generator/              # 生成器源码
├── client-id               # Machine Identity Client ID
├── client-secret           # Machine Identity Client Secret（注意权限）
└── secrets/                # 生成的 secrets 文件目录（自动创建）
```

**docker-compose.yml**：
```yaml
services:
  infisical-agent:
    image: infisical/cli:latest
    container_name: infisical-agent
    restart: always
    command: agent --config=/config/config.yaml
    volumes:
      - ./config.yaml:/config/config.yaml:ro
      - ./client-id:/config/client-id:ro
      - ./client-secret:/config/client-secret:ro
      - ./secrets:/infisical-secrets
```

**config.yaml**（服务配置）：
```yaml
# Infisical 项目 ID
project_id: "<your-project-id>"

# 环境 (dev/staging/prod)
environment: "prod"

# 轮询间隔
polling_interval: "300s"

# 服务列表 - 每个服务对应 Infisical 中的一个文件夹
services:
  - vaultwarden
  - postgres
  - nginx
```

**生成 config.yaml**：
```bash
cd infisical-agent
./generate.exe
```

### 第三步：修改业务服务配置

以 vaultwarden 为例，在服务目录下创建符号链接：

```bash
cd vaultwarden

# 备份原有的 .env 文件
cp .env .env.backup

# 创建符号链接指向 Infisical 生成的 secrets 文件
ln -sf ../infisical-agent/secrets/vaultwarden.env .env
```

同时在 `docker-compose.yml` 中添加 `env_file`：

```yaml
services:
  vaultwarden:
    env_file:
      - .env
    environment:
      DOMAIN: https://password.yewyard.cn  # 固定值直接写
      SIGNUPS_ALLOWED: false
      LOG_LEVEL: ${LOG_LEVEL:-warn}  # 可以用默认值语法
    # ...其他配置
```

这样 secrets 会通过两种方式生效：
- **符号链接的 `.env`**：让 `${VAR}` 变量替换和默认值语法正常工作
- **`env_file: .env`**：确保所有变量都注入到容器环境中

### 第四步：验证并清理

迁移完成并验证服务正常运行后，可以删除备份文件 `.env.backup`。

> **注意**：`.env` 现在是符号链接，不要删除它，否则服务将无法读取环境变量。

---

## 添加新服务（快速参考）

只需 3 步：

1. **Infisical 中**：创建文件夹 `/<服务名>`，添加 secrets
2. **config.yaml 中**：在 `services` 列表添加服务名，运行 `./generate.exe`
3. **服务目录中**：
   ```bash
   cd <服务目录>
   cp .env .env.backup  # 备份原文件（如果存在）
   ln -sf ../infisical-agent/secrets/<服务名>.env .env
   ```
   并在 `docker-compose.yml` 中添加 `env_file: .env`

---

## 验证步骤

1. **启动 Infisical Agent**
   ```bash
   cd infisical-agent
   docker compose up -d
   ```

2. **检查 secrets 文件是否生成**
   ```bash
   cat infisical-agent/secrets/vaultwarden.env
   ```

3. **重启业务服务**
   ```bash
   cd vaultwarden
   docker compose up -d
   ```

4. **验证环境变量注入成功**
   ```bash
   docker exec vaultwarden printenv | grep ADMIN_TOKEN
   ```

---

## 注意事项

1. **client-secret 文件权限**：建议设置为 `chmod 600`
2. **启动顺序**：必须先启动 infisical-agent，等 secrets 文件生成后再启动其他服务
3. **热更新**：Agent 每 5 分钟轮询一次，但容器不会自动重启
4. **.gitignore**：记得忽略 `client-id`、`client-secret` 和 `secrets/` 目录
