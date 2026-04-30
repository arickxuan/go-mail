package middleware

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// AdminTokenAuth 管理员 Token 认证中间件
func AdminTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从环境变量获取预期 token
		expectedToken := os.Getenv("X_ADMIN_TOKEN")

		// 如果环境变量未配置，允许通过（根据需求调整）
		if expectedToken == "" {
			c.Next()
			return
		}

		// 从 Header 中获取 token（注意：Gin 会自动规范化 Header 名称）
		// 客户端可以传 X-ADMIN-TOKEN 或 X-Admin-Token
		token := c.GetHeader("X-Admin-Token")

		// 如果 token 为空，也尝试获取原始 key
		if token == "" {
			token = c.GetHeader("X-ADMIN-TOKEN")
		}

		// 验证 token 是否匹配
		if token != expectedToken {
			// 重定向到登录页面
			//c.Redirect(http.StatusFound, "/login")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort() // 停止后续处理
			return
		}

		// Token 验证通过，继续处理请求
		c.Next()
	}
}
