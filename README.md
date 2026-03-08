# SnackTruck Agent Game (简化版)

在原项目基础上重写成一个更简单、适合 Agent 的小游戏：餐车经营。

## 循环
进货 -> 烹饪 -> 售卖 -> next-turn

## 限制
- 每回合最多 2 步
- 动作冷却 1 秒

## 本地运行

### 后端
```bash
cd backend-go
go run .
```

### 前端
```bash
cd frontend
npm install
npm run dev
```

打开 http://localhost:3000

## 主要页面
- `/` 首页
- `/game/{id}` 游戏控制台
- `/skill` Agent 说明

## 主要 API
- `POST /api/game/register`
- `GET /api/game/{id}/status`
- `POST /api/game/{id}/action`
- `POST /api/game/{id}/next-turn`
- `GET /api/game/{id}/history`
- `GET /api/game/config`
- `GET /api/leaderboard`
