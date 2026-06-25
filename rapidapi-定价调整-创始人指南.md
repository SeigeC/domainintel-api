# DomainIntel RapidAPI 定价调整 — 创始人操作指南

> 2026-06-25 | 评估实验室审计后调价

## 新定价

| 套餐 | 旧价 | 新价 | 月配额 | RPS |
|------|------|------|--------|-----|
| BASIC | Free | Free | 500 | 1 |
| PRO | $9/mo | **$25/mo** | 10,000 | 10 |
| ULTRA | $29/mo | **$75/mo** | 100,000 | 50 |
| MEGA | — | **$150/mo** | 500,000 | 100 |

## 操作步骤

### 1. 登录 RapidAPI Provider Dashboard

打开 https://my.rapidapi.com/studio

### 2. 进入 DomainIntel API 的 Plans 页面

- 左侧菜单找到 **My APIs** → 点击 **DomainIntel**
- 进入 **Plans** 标签

### 3. 修改现有套餐价格

**PRO 套餐**：
1. 点击 PRO plan 的 Edit
2. 月价格从 `$9` 改为 `$25`
3. 保存

**ULTRA 套餐**：
1. 点击 ULTRA plan 的 Edit
2. 月价格从 `$29` 改为 `$75`
3. 保存

### 4. 新增 MEGA 套餐

1. 点击 **Add Plan**
2. Name: `MEGA`
3. Price: `$150` / month
4. Quota: `500,000` requests/month
5. Rate limit: `100` requests/second
6. Endpoint access: 全部端点
7. 保存

### 5. 验证

- 检查 Plans 页面显示：BASIC Free | PRO $25 | ULTRA $75 | MEGA $150
- 确认每个套餐的配额和 RPS 正确

## 已完成（AI 端）

- ✅ index.html（GitHub Pages 落地页）定价更新
- ✅ README.md 定价表更新
- ✅ docs/index.html（API 文档页）定价更新 + 新增 MEGA
- ✅ rapidapi-上架指南.md 定价表 + 套餐配置更新
- ✅ bot-api-mvp-plan.md 定价表更新
- ✅ marketing-plan.md 两处定价引用更新
- ✅ 代码已 commit 并 push 到 `git@github.com:SeigeC/domainintel-api.git`

## 预估收入（RapidAPI 抽 25% 后）

| 套餐 | 标价 | 实收/月 |
|------|------|---------|
| PRO | $25 | ~$17.73 |
| ULTRA | $75 | ~$53.73 |
| MEGA | $150 | ~$109.73 |
