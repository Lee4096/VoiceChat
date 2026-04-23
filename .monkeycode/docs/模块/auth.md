# 认证服务模块

## 概述

认证服务模块 (`internal/auth/`) 提供用户认证和授权功能，支持 JWT Token、密码认证和 OAuth2 (GitHub/Google)。

## 组件

### JWT Service (jwt.go)

JWT Token 生成和验证。

**功能**:
- 生成 Access Token
- 验证 Token 有效性
- 刷新 Token

**实现**:
```go
type JWTService struct {
    secret     string
    expiration time.Duration
}

func (s *JWTService) GenerateToken(userID, email string) (string, error)
func (s *JWTService) ValidateToken(token string) (*Claims, error)
func (s *JWTService) RefreshToken(token string) (string, error)
```

### Password Service (password.go)

密码注册和登录。

**功能**:
- 用户注册 (密码 + 邮箱)
- 用户登录验证
- 密码哈希 (bcrypt)

**实现**:
```go
type PasswordService struct {
    userService *user.Service
}

func (s *PasswordService) Register(ctx context.Context, email, password, name string) (string, error)
func (s *PasswordService) Login(ctx context.Context, email, password string) (string, error)
func (s *PasswordService) UserExists(ctx context.Context, email string) (bool, error)
```

### OAuth2 Service (oauth2.go)

OAuth2 第三方认证。

**支持**:
- GitHub
- Google

**流程**:
1. 获取授权 URL
2. 处理回调
3. 创建/更新用户
4. 生成 JWT Token

**实现**:
```go
type OAuth2Service struct {
    githubConfig OAuth2Config
    googleConfig OAuth2Config
}

func (s *OAuth2Service) GitHubLoginURL() string
func (s *OAuth2Service) GitHubCallback(ctx context.Context, code string) (*OAuthUser, error)
func (s *OAuth2Service) GoogleLoginURL() string
func (s *OAuth2Service) GoogleCallback(ctx context.Context, code string) (*OAuthUser, error)
```

## 数据类型

```go
// OAuthUser OAuth 用户信息
type OAuthUser struct {
    Email string
    Name  string
    ID    string  // OAuth provider user ID
}

// OAuth2Config OAuth2 配置
type OAuth2Config struct {
    ClientID     string
    ClientSecret string
    CallbackURL  string
}
```

## 错误类型

| 错误 | 说明 |
|------|------|
| ErrInvalidCredentials | 无效的凭据 |
| ErrUserExists | 用户已存在 |
| ErrOAuthFailed | OAuth 认证失败 |
| ErrTokenInvalid | Token 无效 |
| ErrTokenExpired | Token 过期 |

## 安全考虑

1. **密码安全**
   - bcrypt 哈希 (cost 12)
   - 最小密码长度 6 字符

2. **Token 安全**
   - 短期过期 (24h)
   - 安全存储
   - 定期刷新

3. **OAuth 安全**
   - State 参数验证
   - Code 一次性使用
   - HTTPS 要求

## 配置

```yaml
jwt:
  secret: "your-secret-key"
  expiration: 86400  # 24小时

oauth2:
  github:
    client_id: ""
    client_secret: ""
    callback_url: "http://localhost:8080/api/v1/auth/callback/github"
  google:
    client_id: ""
    client_secret: ""
    callback_url: "http://localhost:8080/api/v1/auth/callback/google"
```
