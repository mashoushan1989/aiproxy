# 模型同步与路由分区方案 v3 (实施版)

> **版本**: v3-shipped — 2026-04-25
> **定位**: 在最小架构改动前提下解决三个问题：
>   (1) sync 擦除 autodiscover/虚拟/手工模型；
>   (2) PPIO 与 Novita 模型同步互相干扰；
>   (3) 海外节点请求漂移到 PPIO 渠道
> **前身**: v3-minimal 草案只覆盖 (1)；实施时在同一 PR 内追加了 W2/W3 路由层改动，覆盖 (2)(3)

---

## 1. 要解决的问题

### 1.1 同步擦除 (W1)

线上 PPIO/Novita sync 用 Replace 策略覆盖 `channels.models`，把原本可服务的模型从渠道列表擦除。**确认场景**：

- 闭源同步生图模型：`gemini-3.1-flash-image-*`、`nano-banana-*`、`veo-3.1-*` 等——PPIO `/product/multimodal-model/list` 不返回，但模型在 `/v3/{model}` sync 端点可用
- 自动发现注册模型——首次成功请求后写入 `model_configs` + 追加到 `channel.Models`，下次 sync 全量 Replace 再次擦除
- 虚拟模型（`ppio-web-search` 等）——靠代码硬编码注入兜住，每次 sync 都要重新拼
- 手工添加模型——管理员 UI 编辑 channel.Models，下次 sync 仍被覆盖

共同点：**不是上游 sync API 返回的，但它们是真实可用的**。

### 1.2 PPIO/Novita 互相干扰 (W1 同方案)

`ShouldSkipOwnership` + `ownerPriority` 机制让 PPIO 静默覆盖 Novita 写过的 ModelConfig。结果：Novita 的端点声明、价格、限速被 PPIO 抹平，Novita 那侧的实际能力丢失。

### 1.3 海外节点路由漂移 (W2 + W3)

`getRandomChannel` 在 multi-set 路径下把所有 set 的 channel **合并到同一个 `channelMap`** 做 weighted random。配合 `GetAvailableSets` 在海外节点返回 `[overseas, default]`，导致海外请求约 50% 概率被路由到 default set 的 PPIO channel —— 与"海外不漂移"目标直接冲突。

---

## 2. 核心洞察

**两个独立但同一 PR 的修复**：

1. **W1 — sync 字段化**：通过 `synced_from` 字段区分"自己写的"vs"别人写的"模型，sync 只管自己。同时引入 merge 语义保护 channel.Models。
2. **W2/W3 — 路由严格分区**：新增 `STRICT_NODE_SET` env，让海外节点可选择"只用 overseas，不 fallback default"。

两者互补：W1 让两边的 sync 在数据层互不干扰；W2/W3 让两边的请求在路由层互不漂移。

---

## 3. 数据模型改动

`model_configs` 表新增 2 列，完全 additive (GORM AutoMigrate 自动加列)：

```sql
-- GORM AutoMigrate 等价的 DDL（PG/SQLite 通用）
ALTER TABLE model_configs ADD COLUMN synced_from   VARCHAR(32);
ALTER TABLE model_configs ADD COLUMN missing_count INT NOT NULL DEFAULT 0;
CREATE INDEX idx_model_configs_synced_from ON model_configs(synced_from);
```

### 3.1 字段语义

| 字段 | 值 | 含义 |
|---|---|---|
| `synced_from` | `'ppio'` | 由 PPIO sync（V1/V2/multimodal 任一路径）写入 |
| | `'novita'` | 由 Novita sync 写入 |
| | `''` (NULL) | **非 sync 来源**——autodiscover / virtual / manual / yaml overlay |
| `missing_count` | `0..N` | 连续 sync miss 计数（仅对 `synced_from = ''` 不为空的行累积） |

**注**：相比草案的 4 个标签 (`ppio_v2` / `ppio_multimodal` / `novita_v2` / `novita_multimodal`)，实施版简化为**每 provider 一个标签**。原因：
- 一次 ExecuteSync 同时跑 chat 和 multimodal 两条上游 API，用同一标签便于在 ExecuteSync 末尾**一次性**调用 `MarkMissingForSync` 处理 union(chat, multimodal)，避免子标签之间互相误判。
- 简化心智负担。

### 3.2 字段使用原则

1. `synced_from` 由**写入方**决定：sync 写就带 sync 标签；autodiscover / manual API / 虚拟注入**保持空**
2. `synced_from = ''` 的行被 sync **永不触碰**：不改配置、不清除、不降级。这由 `synccommon.CanSyncOwn(existing, mine)` 强制
3. `synced_from` 非空的行：sync 负责维护生命周期——本次看到重置 missing_count=0，没看到累加；超过 `SyncMissingThreshold (=7)` 从 channel.Models 剔除（**model_configs 行保留**，admin 可手动恢复）

### 3.3 启动时的一次性 backfill

`migrateModelConfigSyncedFrom` 在 `EnterpriseAutoMigrate` 启动时跑，**幂等**：

```sql
-- 仅给 synced_from='' 的行打标签
UPDATE model_configs SET synced_from = 'ppio'
WHERE owner = 'ppio' AND (synced_from IS NULL OR synced_from = '');

UPDATE model_configs SET synced_from = 'novita'
WHERE owner = 'novita' AND (synced_from IS NULL OR synced_from = '');
```

其它 owner（`deepseek`、`anthropic`、`google` 等）保持 `synced_from=''` —— 它们是手工/yaml 录入，sync 不该管。

---

## 4. Sync 行为改动 (W1)

### 4.1 ModelConfig 写入：四处全部带 SyncedFrom

PPIO 4 处 + Novita 4 处共 8 处 ModelConfig 创建/更新（V1 chat / V2 chat / multimodal create / multimodal update），全部：

- 显式赋值 `SyncedFrom = synccommon.SyncedFromPPIO` (或 `SyncedFromNovita`)
- 重置 `MissingCount = 0`（看到了上游，重置计数）
- 操作前用 `synccommon.CanSyncOwn(existing.SyncedFrom, mine)` 守卫——禁止覆盖其它 sync 的行或非 sync 行

### 4.2 channel.Models 改 merge 语义

8 处 `channels[i].Models = upstreamList` 全部替换为：

```go
channels[i].Models = synccommon.MergeChannelModels(
    db,
    syncedFrom,                   // "ppio" or "novita"
    upstreamSubsetForChannel,     // 本次本类型上游列表
    channels[i].Models,           // 当前列表
)
```

`MergeChannelModels` 输出 = `upstreamSubset ∪ (本 sync 自己的、aging<7 的) ∪ (非本 sync 拥有的，preserved)`。

### 4.3 missing_count 维护

每次 ExecuteSync 末尾、Channel 更新前调用一次：

```go
synccommon.MarkMissingForSync(db, syncedFrom, union_of_chat_and_multimodal)
```

它对 `synced_from=mine AND model NOT IN union` 的行执行 `missing_count = missing_count + 1`。已经在创建/更新分支重置过的行不受影响。

仅在 `len(allModels) > 0` 时调用——上游全失败时跳过，避免 spurious decay。

### 4.4 删除 ownership priority

删除：
- `synccommon.ShouldSkipOwnership`
- `synccommon.CanClaimOwnership`
- `synccommon.ownerPriority`
- 所有调用点（PPIO 4 处、Novita 4 处）

替代物：`CanSyncOwn(existing, mine)` —— 严格按 `synced_from` 字段判定，不再有跨 provider 抢占。

### 4.5 sync delete 改用 synced_from 过滤

```diff
- WHERE model = ? AND owner = ?
+ WHERE model = ? AND synced_from = ?
```

确保即使 owner=ppio 的 autodiscover 行（synced_from='') 不会被 sync 误删。

### 4.6 Compare/Diff 函数同步

`ComparePPIOModelsV2` / `CompareNovitaModelsV2` 的判定从 `Owner` 改为 `SyncedFrom`：

- "Shared" (跨 sync 共享，sync 不动): `localModel.SyncedFrom != mySyncedFrom`
- "Delete 候选" (本 sync 拥有但上游没有): `mc.SyncedFrom == mySyncedFrom`

`diagnostic.go` 的 localCount 也按 `synced_from` 计数。

---

## 5. 跨节点并发保护 (W1 安全网)

国内+海外节点共享同一 PG（WireGuard 隧道）。两节点都在跑 PPIO 调度器 + Novita 调度器，原代码只有进程内 `syncMu` 互斥。

**新增**：`synccommon.AcquireSyncLock(db, syncedFrom)` / `ReleaseSyncLock` 使用 `pg_try_advisory_lock` 跨节点互斥：

- 锁键固定：`SyncedFromPPIO=0x4149505052494F00`、`SyncedFromNovita=0x4149504E4F564954`
- ExecuteSync 入口处 try-lock；失败返回 `ErrSyncInProgress`
- defer 释放；session 结束自动释放（崩溃自愈）
- SQLite 路径 no-op（开发/单测使用 `syncMu` 即可）

---

## 6. Autodiscover 改动 (W1 配套)

四处 `model.DB.Save(&mc)` 调用点（PPIO 2 + Novita 4）的 ModelConfig 字面量**显式不设** `SyncedFrom` 字段（保持零值 `""`），并加注释说明语义。

---

## 7. 路由层改动 (W2 + W3)

### 7.1 新增 env

```go
// core/common/config/config.go
var strictNodeSet = env.Bool("STRICT_NODE_SET", false)
func GetStrictNodeSet() bool
func SetStrictNodeSet(bool)  // 测试用
```

### 7.2 GetAvailableSets 严格模式 (W3)

```go
// core/model/cache.go
if nodeSet := config.GetNodeChannelSet(); nodeSet != "" {
    if nodeSet == ChannelDefaultSet {
        return []string{ChannelDefaultSet}
    }
    if config.GetStrictNodeSet() {
        return []string{nodeSet}              // ← strict: 只本 set
    }
    return []string{nodeSet, ChannelDefaultSet}  // soft (default): 保留 fallback
}
```

### 7.3 getChannelWithFallback 严格模式 (W2)

multi-set 路径下，strict=true 且 primary set (`i==0`) 出现 `ErrChannelsNotFound` 或 `ErrChannelsExhausted` 时**硬失败**，不再 fallback 到 next set。

### 7.4 Shadow 日志

strict=false（默认软 fallback）路径下，**记录**"如果开 strict 这次就要拒绝"的事件，每 (model, set, reason) 元组每分钟最多 1 条 WARN：

```
shadow_strict_would_reject model=pa/claude-opus-4-7 set=overseas reason=not_found_soft_fallback
```

运维在切 strict 之前观察 24h，如果日志稀疏即可放心切。带速率限制避免 disk fill。

---

## 8. 实施清单

### 8.1 触及文件

| 文件 | 改动 |
|---|---|
| `core/model/modelconfig.go` | 加 `SyncedFrom` + `MissingCount` 字段 |
| `core/enterprise/synccommon/merge.go` (新) | `MarkMissingForSync` / `MergeChannelModels` / `CanSyncOwn` / `AcquireSyncLock` / `ReleaseSyncLock` |
| `core/enterprise/synccommon/merge_test.go` (新) | 10+ 单测 |
| `core/enterprise/synccommon/synccommon.go` | 删除 `ownerPriority` / `CanClaimOwnership` / `ShouldSkipOwnership` |
| `core/enterprise/ppio/sync.go` | 4 处 ModelConfig 写改 SyncedFrom；merge 语义；MarkMissingForSync；advisory lock；delete WHERE 改 synced_from |
| `core/enterprise/novita/sync.go` | 同上 |
| `core/enterprise/ppio/autodiscover.go` | 显式 SyncedFrom='' 注释 |
| `core/enterprise/novita/autodiscover.go` | 同上 |
| `core/enterprise/ppio/diagnostic.go` | localCount + Shared/Delete 判定改 synced_from |
| `core/enterprise/novita/diagnostic.go` | 同上 |
| `core/enterprise/models/migrate.go` | `migrateModelConfigSyncedFrom` 一次性 backfill |
| `core/common/config/config.go` | 加 `STRICT_NODE_SET` env |
| `core/model/cache.go` | `GetAvailableSets` 加 strict 分支 |
| `core/controller/relay-channel.go` | strict 分支 + shadow 日志（带速率限制） |
| `core/enterprise/{ppio,novita}/sync_channels_test.go` | 适配 merge 语义新预期 |

### 8.2 测试

- 单测全绿（`go test -tags enterprise ./...`）
- merge 函数 8+ 单测覆盖：upstream new / aging / threshold / cross-sync preserve / non-sync preserve / no config row 等
- 现有 sync_channels_test 已更新预期：unowned 模型现在会被保留（新行为）

---

## 9. Rollout

### 9.1 部署节奏（共享 PG，独立 Redis）

| 阶段 | 操作 | 风险 |
|---|---|---|
| **T-1 晚** | 海外节点 `DISABLE_NOVITA_AUTO_SYNC=true` + 重启（避免窗口期写污染） | 0 |
| **T+0** | 国内节点 `deploy.sh`（GORM AutoMigrate 自动加 `synced_from` / `missing_count` 列；`migrateModelConfigSyncedFrom` 自动 backfill） | 极低 |
| **T+0** | 海外节点 `deploy.sh` | 极低 |
| **T+0+30min** | 手动 trigger 一次 PPIO sync 验证 | 0 |
| **T+0+60min** | 手动 trigger 一次 Novita sync 验证；`DISABLE_NOVITA_AUTO_SYNC=` 取消禁用 | 0 |
| **T+24h~T+48h** | 观察 `shadow_strict_would_reject` 日志数量与涉及 model | 0 |
| **T+灰度日** | 海外节点 `STRICT_NODE_SET=true` + 重启 | 中（仅在事先验证 `overseas` channel 数据完整时） |

### 9.2 strict 切换前必跑 SQL

```sql
-- overseas 渠道声明的所有模型，是否都有对应 ModelConfig 行
SELECT DISTINCT unnest(models) AS model
FROM channels
WHERE sets::text LIKE '%overseas%' AND status = 1
EXCEPT
SELECT model FROM model_configs;
```

返回 0 行才能切。否则海外用户对缺失的模型会立即 404。

### 9.3 回滚

- **代码**：`bash scripts/deploy.sh --rollback`，~5 分钟
- **数据**：synced_from 字段是 additive，旧代码完全可读（GORM SELECT 忽略未知字段）。**无需回滚 schema**
- **strict mode**：删 env + 重启，~5 分钟
- **advisory lock**：session 结束自动释放，无需手动清理

---

## 10. 能解决与不能解决

### ✅ 解决

- 闭源多模态模型不再被 sync 擦除
- autodiscover/virtual/manual 模型永久保留
- PPIO 和 Novita sync 互不覆盖（`CanSyncOwn` 严格守卫）
- 海外节点请求绝不漂移到 PPIO（strict mode 启用后）
- 删除 ownership priority 黑魔法
- 跨节点并发 sync 由 PG advisory lock 串行化
- shadow 日志为 strict 切换提供决策依据

### ❌ 不解决（已知遗留，留待未来）

| 问题 | 何时处理 |
|---|---|
| ModelConfig 仍是全局单行（PPIO 和 Novita 同名模型共享 price/endpoints） | 真出现两边发散需求时，做 (model, set) 复合主键升级 |
| Logs 表无 set 列（无法按 region 分账） | 财务提需求时（独立 PR） |
| GroupModelConfig 不带 set 维度（override 跨 set 抹平） | 同上 |
| Admin UI 不展示 synced_from / missing_count 状态 | UI 迭代时 |
| `applyYAMLConfigToModelConfigCache` 不显式带 set | YAML overlay 增强时 |

---

## 11. 总结

- **2 字段**：`synced_from` + `missing_count`
- **1 env**：`STRICT_NODE_SET`
- **1 共享函数文件**：`synccommon/merge.go` (~250 行 + 测试 ~250 行)
- **6 处补丁删除**：ownership priority 整套机制
- **15 个文件改动**：~600 行净增
- **回滚成本**：~5 分钟
- **生产风险**：低（schema 全 additive、env 默认 false 保持 soft fallback）

**核心口号**：**sync 只管自己写过的；路由严格留在自己 set 里。**

---

## 附录 — 未来演进路线

若以下场景出现，按需独立 PR 推进：

1. **接入第 3 家 provider** → 加新 `SyncedFromXxx` 常量
2. **PPIO/Novita 同名模型 price 真发散** → 升级 `(model, set)` 复合主键（参考早期 v3-composite-key 评审记录）
3. **Billing 按 region 分账** → logs 加 set 列
4. **autodiscover 4xx 探测** → 加 `attempt_discover` hook
5. **运维主动探活** → 加 `scheduled_probe` 任务

每条都可作为独立小 PR 推进，不需要一次性大重构。

---

## 附录 — 相关 Task / 历史记录

- 2026-04-24 评估了 `(model, set)` 复合主键方案，因端点独立性需求暂时不强（PPIO/Novita 端点几乎一致）+ 单 PR 风险面过大（6 个 P0、17 个 P1）而推迟
- 路径选择为 v3-shipped (W1 + W2 + W3)，本文件即该方案的实施记录
