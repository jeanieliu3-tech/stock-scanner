# 项目概览

**波段趋势共振助手** — 基于 Vue 3 + Go 的股票波段趋势分析与扫描工具，提供双引擎扫描、个股诊断、持仓管理等功能。

## 技术栈

- **Frontend**: Vue 3 + TypeScript + Vite + Tailwind CSS 4 + Lucide Icons
- **Backend**: Go (Gin) + 新浪行情API + 腾讯K线数据
- **包管理**: pnpm (前端), go mod (后端)

## 目录结构

```
├── frontend/                # Vue 3 前端
│   ├── src/
│   │   ├── api/             # API 请求层
│   │   ├── assets/          # 静态资源
│   │   ├── components/      # 组件 (AppLayout, StockDetailSheet)
│   │   ├── composables/     # 组合式函数
│   │   ├── router/          # Vue Router
│   │   ├── types/           # TypeScript 类型定义
│   │   └── views/           # 页面视图 (Home, Scan, Position, Settings)
│   ├── index.html
│   ├── vite.config.ts
│   └── package.json
├── backend/                 # Go 后端
│   ├── handlers/            # HTTP 处理器
│   ├── models/              # 数据模型
│   ├── services/            # 业务逻辑
│   ├── main.go              # 入口
│   └── go.mod
├── scripts/                 # 构建/启动脚本
└── .coze                    # 项目配置
```

## 构建与运行命令

- **开发**: `bash scripts/prepare.sh && bash scripts/dev.sh` (Go后端3000端口 + Vite前端5000端口)
- **构建**: `bash scripts/build.sh`
- **生产运行**: `bash scripts/start.sh`

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/stock/market | 大盘状态 |
| GET | /api/stock/quotes?codes=xx | 批量报价 |
| POST | /api/stock/scan-dual-engine-fast | 双引擎快速扫描 |
| POST | /api/stock/scan-core-satellite | 核心-卫星扫描 |
| POST | /api/stock/diagnose | 个股诊断 |
| GET | /api/stock/watchlist | 关注列表 |
| POST | /api/stock/watchlist/add | 添加关注 |
| POST | /api/stock/watchlist/remove | 移除关注 |
| GET | /api/stock/watchlist/scan | 扫描关注列表 |
| GET | /api/stock/detail/:code | 股票详情(MACD/BOLL) |
| GET | /api/stock/scores?codes=xx | 批量评分 |
| GET | /api/stock/search?keyword=xx | 搜索股票 |
| POST | /api/stock/scan-all | 全A股扫描(约5000支) |
| GET | /api/stock/rank?page=&pageSize=&sortBy=&order=&filter= | 全A股排名(分页+筛选,filter支持: industry_xxx,change_gt_N,score_gte_N) |
| GET | /api/health | 健康检查 |

## 开发规范

- 前端使用 Vue 3 Composition API + `<script setup>` 语法
- 后端 Go 代码遵循标准 Go 项目布局
- 所有 API 响应格式: `{ code: number, msg: string, data: T }`
- 中国股市配色: 红涨绿跌
- 仅使用 pnpm 管理 Node.js 依赖
