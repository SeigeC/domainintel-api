# DomainIntel API — RapidAPI Provider 上架指南

里程碑 A-M3。本指南引导你从零完成 RapidAPI Provider 注册到 API 上架。

> **状态：已完成上架 (2026-06-24)**
> - Provider: DomainIntel (org id: 12086844)
> - API: DomainIntel (api slug: domainintel)
> - Base URL: https://domainintel.onrender.com
> - 5 端点: Health/GET, RDAP Lookup/GET, DNS Lookup/GET, Certificates/GET, Bulk Lookup/POST
> - 3 套餐: BASIC $0 (500/月), PRO $25 (10,000/月), ULTRA $75 (100,000/月), MEGA $150 (500,000/月)
> - 可见性: Public
> - Proxy Secret: 87ff4840-6fc9-11f1-bd13-074a384708ef
> - 待办: 在 Render 设置 RAPIDAPI_PROXY_SECRET 环境变量

## 1. Provider 注册引导

1. 用已注册的开发者账号登录 [rapidapi.com](https://rapidapi.com)。
2. 访问 [rapidapi.com/provider](https://rapidapi.com/provider) 进入 Provider Dashboard。首次进入会提示创建 Provider。
3. 点击 **Create Provider / Become a Provider**，填写：
   - **Provider Name**：`DomainIntel`（或你的个人品牌名）
   - **Description**：一句话说明，如"提供域名 RDAP、DNS、证书透明度查询的情报 API"
   - **Logo**：建议用简单文字 logo（白底深色字，512x512 PNG）。无需设计图，Figma/Canva 拼几个字母即可。
4. 提交后进入 Provider Dashboard 主页，左侧菜单包含 APIs、Analytics、Billing 等。

## 2. API 创建与配置

1. 在 Provider Dashboard 点击 **Add New API**。
2. 填写基本信息：
   - **API Name**：`DomainIntel API`
   - **Short Description**：`Domain intelligence API — RDAP, DNS, certificate transparency lookups`
   - **Category**：`Data`（也可选 `Tools`）
   - **API Base URL**：`https://domainintel.onrender.com`（已部署验证通过）
3. 保存后进入 API 详情页，开始添加端点。

## 3. 端点映射

在 API 详情页 **Endpoints** 标签逐个添加。下表对应 `openapi.yaml` 定义：

| RapidAPI 端点名 | 方法 | 路径 | 描述 | 参数 |
|---|---|---|---|---|
| Health | GET | /v1/health | 健康检查，不限流 | 无 |
| RDAP Lookup | GET | /v1/rdap/{domain} | RDAP 域名注册信息查询 | path: `domain`（必填） |
| DNS Lookup | GET | /v1/dns/{domain} | DNS over HTTPS 记录查询 | path: `domain`（必填）；query: `type`（可选，枚举 A/AAAA/NS/CNAME/SOA/MX/TXT/CAA，默认 A） |
| Certificates | GET | /v1/certificates/{domain} | crt.sh 证书透明度查询 | path: `domain`（必填）；query: `limit`（可选，1-500，默认 50）、`match`（可选，枚举 wildcard） |
| Bulk Lookup | POST | /v1/bulk | 批量多类型查询（最多 20 域名） | body: `domains[]`（必填，≤20）、`types[]`（必填，枚举 rdap/dns/certificates）、`dns_type`（可选，默认 A） |

每个端点添加时配置返回示例 JSON（从 openapi.yaml 的 example 复制），便于 RapidAPI 自动生成文档。

## 4. 套餐配置

在 API 详情页 **Pricing** 标签添加三档套餐：

| 套餐 | 价格 | 月配额 | RPS | 可用端点 |
|---|---|---|---|---|
| Free | $0 | 500 次/月 | 1 | rdap + dns |
| Pro | $25/月 | 10,000 次/月 | 10 | 全端点 + bulk |
| Business | $75/月 | 100,000 次/月 | 50 | 全功能 + 优先缓存 |
| Mega | $150/月 | 500,000 次/月 | 100 | 全功能 + 优先缓存 + 优先支持 |

每个套餐设置方法：
1. 点击 **Add Pricing Plan**，命名 Free/Pro/Business，填价格。
2. **Quotas**：设置 Monthly Requests（500 / 10000 / 100000 / 500000）。
3. **Rate Limits**：设置 Requests Per Second（1 / 10 / 50 / 100）。
4. **Endpoints Access**：勾选该套餐可调用的端点（Free 仅勾 rdap+dns）。
5. 保存后启用。

## 5. 部署前准备清单

API 必须先部署到公网才能上架测试。

**部署选项**：Oracle ARM（永久免费层）/ Render / Koyeb。三者都支持 Docker 部署，仓库已含 `Dockerfile`。

**部署后操作**：

1. 更新 RapidAPI API Base URL 为真实部署地址（如 `https://domainintel.onrender.com`）。
2. 在 RapidAPI Provider Dashboard 的 API 设置中找到 **Proxy Secret**，复制该值。
3. 在部署环境设置环境变量 `RAPIDAPI_PROXY_SECRET=<复制的值>`。
4. API 已内置 Proxy Secret 验证中间件（见 `proxy_secret.go`），环境变量为空时跳过验证（开发模式），设置后强制校验 `X-RapidAPI-Proxy-Secret` header。

核心代码片段：

```go
func ProxySecretMiddleware() func(http.Handler) http.Handler {
    secret := os.Getenv("RAPIDAPI_PROXY_SECRET")
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if secret == "" {
                next.ServeHTTP(w, r) // 开发模式跳过
                return
            }
            provided := r.Header.Get("X-RapidAPI-Proxy-Secret")
            if provided == "" ||
                subtle.ConstantTimeCompare([]byte(provided), []byte(secret)) != 1 {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusUnauthorized)
                _, _ = w.Write([]byte(`{"error":{"code":401,"message":"unauthorized","type":"unauthorized"}}`))
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

5. 用 curl 测试连通性：`curl https://domainintel.onrender.com/v1/health` 应返回 `{"status":"ok"}`（已验证通过）。

## 6. 上架后验证步骤

1. 在 RapidAPI **Endpoints** 标签用内置测试器逐个调用 5 个端点，确认 200 响应。
2. 连续快速调用超过 RPS 限额，确认返回 429 + `Retry-After` header。
3. 触发错误场景（无效域名、超限 limit），确认返回 `{"error":{"code":...,"type":"..."}}` 格式。
4. 在 Provider Dashboard **Analytics** 查看配额计数是否随调用递增。
5. 切换不同套餐订阅测试端点访问权限是否正确拦截。

## 7. RapidAPI 注意事项

- **抽成**：RapidAPI 抽 25% + 2.9% + $0.30/笔。$25 套餐实收约 $17.73，$75 套餐实收约 $53.73，$150 套餐实收约 $109.73。
- **结算**：净 60 天，起付门槛 $100。前期收入会沉淀较久，做好现金流预期。
- **平台风险**：Nokia 收购后重心偏移，平台投入存在不确定性。建议同步建独立渠道（自建落地页 + Stripe 直收），RapidAPI 作为获客入口而非唯一渠道。
- **退款政策**：RapidAPI 主导退款裁决，Provider 无最终决定权。明确各端点的预期行为可减少纠纷。
- **下架影响**：暂停 API 会导致已订阅用户立即无法调用，触发退款和差评。维护前先用 Provider 公告通知用户。
