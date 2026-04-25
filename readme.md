  这是一个 Go + Gin 后端 + 原生 JS
  前端的邮件管理工具，支持 IMAP/POP3/Graph API 三种协议。让我为你写一份详细的
  README，包含 Vercel 部署指南。

  多协议邮件管理工具，支持 IMAP / POP3 / Microsoft Graph API
  三种方式连接邮箱，提供统一的 Web 管理界面。

  功能

  - 多账号管理：添加、编辑、删除邮箱账号
  - 多协议支持：IMAP、POP3、Microsoft Graph API
  - 收件箱浏览：分页查看邮件列表
  - 邮件详情：查看邮件完整内容
  - 批量导入：支持通过文本批量导入账号
  - 主流邮箱：Outlook、Gmail、Yahoo、QQ、163 等

  技术栈

  - 后端: Go + Gin
  - 前端: 原生 JavaScript + CSS（零依赖）
  - 存储: JSON 文件（accounts.json）

  ---
  本地开发

  # 启动服务
  go run main.go

  # 自定义端口
  go run main.go -addr :3000

  访问 http://localhost:8080。

  ---
  部署到 Vercel

  本项目是 Go 应用，通过 Vercel 的 Serverless Functions 运行。

  前置条件

  1. Vercel 账号
  2. Vercel CLI（可选，也可通过网页导入）



  

部署

  方式一：通过 Vercel CLI

  # 安装 CLI
  npm i -g vercel

  # 登录
  vercel login

  # 在项目目录下部署
  vercel

  # 部署到生产环境
  vercel --prod

  方式二：通过网页导入

  1. 访问 vercel.com/new
  2. 导入你的 Git 仓库
  3. Vercel 会自动检测 Go 项目
  4. 在设置中添加 vercel.json 内容
  5. 点击 Deploy

  注意事项

  - 持久化存储：Vercel 的 Serverless 环境不保证文件持久化，accounts.json
  在函数实例被回收后会丢失。建议：
    - 接入外部存储（如 MongoDB、Upstash Redis）
    - 或使用 Vercel KV / Blob Storage
  - 环境变量：如有敏感配置（如默认的 OAuth Client ID），通过 vercel env add 添加
  - 超时：邮件服务器连接可能较慢，Vercel 免费版函数超时为 10s，Pro 版可配置到
  60s。如果遇到超时问题，考虑升级或添加异步处理

  ---
  项目结构

  ├── main.go           # 入口文件
  ├── handlers/         # API 路由处理
  ├── imap/             # IMAP 协议实现
  ├── pop3/             # POP3 协议实现
  ├── graph/            # Microsoft Graph API 实现
  ├── smtp/             # SMTP（预留）
  ├── store/            # 数据存储层
  ├── web/              # 前端静态文件
  │   ├── index.html
  │   ├── app.js
  │   └── style.css
  ├── accounts.json     # 账号数据文件（不要提交到 Git）
  └── go.mod
