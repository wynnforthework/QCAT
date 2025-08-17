# 量化合约自动化交易系统总体方案 v1.0

> 目标：实现“全自动、可持续优化、风险可控”的加密数字货币合约量化交易系统；满足 10 项自动化能力与 Rules/Workflows 要求。后端 Go，本地数据库（PostgreSQL/MySQL 任选）、Redis（可选），前端 React + shadcn/ui。

---

## 一、总体架构

**分层架构**

- **数据层**：本地 DB（OLTP + 历史行情）、对象存储（可选，K线与回测包）、Redis（实时缓存/队列/分布式锁）。
- **服务层（Go）**：
  1. **Market Ingestor**（行情采集）：现货/合约深度、K线、交易、Funding、OI、指数价。
  2. **Exchange Connector**：统一交易所接口（Binance/OKX/Bybit…），下单、撤单、账户、仓位、风险限额、速率控制。
  3. **Strategy Runner**：策略沙箱执行（实时/纸交易/回测），信号→订单流水。
  4. **Portfolio Manager**（PM）：资金与仓位分配、跨策略资本配额、多臂赌博机调度。
  5. **Risk Engine**：硬风控（熔断、限仓、杠杆上限、隔离/全仓、风控前置审批）。
  6. **Optimizer**：参数优化（网格/贝叶斯/CMA-ES/遗传），Walk-Forward & 滚动重训；指标：收益/回撤/波动/成本。
  7. **Orchestrator/Scheduler**：任务编排（周期优化、优胜劣汰、热度扫描、健康检查）。
  8. **Backtester**：事件驱动回测（撮合滑点、资金费、委托簿采样、延迟模型）。
  9. **Hotlist Service**：热门币种识别与白名单化（需手动启用）。
  10. **Monitoring & Audit**：Prometheus 指标、告警、审计日志、可回溯决策链。
  11. **API Gateway**：REST + WebSocket（前端/外部集成）。
- **前端（React + shadcn/ui）**：看板、策略管理、参数调优、风控阈值、工单与审批、热榜与启用面板、日志回放、回测可视化。

**数据流** 行情 → Strategy Runner → 信号 → Risk Engine 审核 → Exchange 下单 → 订单回报/成交 → PM/Runner 状态更新 → DB/Redis → 监控与审计。

---

## 二、关键工作流对齐（对应你的 10 项需求）

1. **盈利未达预期/周期到期 → 自动优化**

- 触发器：`Sharpe<阈值`、`MDD>阈值`、`n 日无新高`、`收益分位<q`、或定时（如每天 UTC 00:10）。
- Orchestrator 生成 `optimizer_task`（含训练/验证区间、目标函数、约束）。
- Optimizer 执行 WFO（Walk-Forward Optimization）→ 产出 `best_params` + 置信区间 + 过拟合检测（Deflated Sharpe、pBO）。

2. **策略自动使用最佳参数**

- 审批流：`best_params` → 风控校验（不超过杠杆/频率/滑点敏感度门限）→ 策略版本化 `strategy_versions` → Canary 分配（10% 资金）→ 达标后 100% 切换。

3. **自动优化仓位**

- PM 以**波动率目标/风险预算**计算目标权重：`w_i = min(w_max, risk_budget_i * target_vol / realized_vol_i)`。
- 结合合约规格与最小变动值，生成**最接近合规**的下单张数。

4. **自动余额驱动的建/减/平仓**

- 监听账户权益变动与未实现盈亏；若保证金占用>门限自动减仓；资金变更触发再平衡。

5. **自动止盈止损**

- 多层次：硬止损（风控）、策略止损（ATR/波动/时间）、移动止盈（Chandelier/Parabolic）、资金曲线止损（回撤阈值）。

6. **周期性自动优化**

- Cron：日频/周频/事件触发；每次优化保存工件与指标曲线，支持回放与回滚。

7. **策略淘汰制（多仓位表现）**

- 以**多臂赌博机**（Thompson/UCB）做资本分配与淘汰：滚动窗口比较 `risk-adjusted return`；末位策略**限时禁用**（如 72 小时），放入冷却池。
- 波动率触发：市场波动率飙升时临时优化参数，避免原策略在极端行情中失效。
- 相关性触发：不同策略之间相关性太高时，自动调整持仓权重，防止集体回撤。

8. **自动增加/启用新策略**

- 新策略接入 `Strategy SDK`，默认纸交易→影子跟单→小额 canary→人工一次性审批后可进入自动化生命周期。

9. **自动调整止盈止损线**

- 将止盈止损参数设为函数：`f(ATR、RV、资金曲线斜率、市场状态 Regime)`，实时滑动更新并持久化版本。

10. **热门币种推荐（需手动启用）**

- 维度：成交额/换手率激增、OI/Funding 异常、波动率分位、社交/新闻分数（可选）；输出候选清单→前端人工启用→纳入白名单。
- 加“风险标签”
不仅推荐热门，还要提示：

价格波动区间

杠杆安全倍数

当前市场情绪（贪婪/恐惧指数）
这样人工审核时能快速判断。
---



**Redis（可选）建议**

- `md:tick:<symbol>` 实时行情；`md:book:<symbol>` 档位；`md:funding:<symbol>`
- `sig:queue` 策略信号队列（List/Stream）；`ord:pending` 待派发订单
- `lock:*` 分布式锁；`rate:<exchange>` 速率限制；`state:pos:<strategy>:<symbol>` 快速态
- `hot:score:<symbol>` 热点分数缓存；`alerts:*` 告警频道（Pub/Sub）

---

## 四、风控与仓位（核心公式）

**1) 市场状态识别（Regime）**

- 波动率分位、趋势因子（Hurst、ADX）、流动性（盘口深度/冲击成本）、Funding/OI 变化。

**2) 目标波动率 + 风险预算**

- 估计实现波动率 `σ_i`（EWMA/parkinson），目标组合波动 `σ*`；
- 单策略权重：`w_i = min(w_max, b_i * σ* / σ_i)`（`b_i` 为策略风险预算系数）；
- 转化为合约张数：`qty = round( w_i * Equity / (σ_i * contract_value * √H) )`，受**杠杆上限**与**保证金**约束。

**3) 多层止盈止损**

- **硬止损**：账户级回撤>阈值、保证金率<阈值 → 全局减仓/平仓。
- **策略止损**：`SL = entry ± k_sl * ATR`；**移动止盈**：`TP = max(prev_TP, price ∓ k_tp * ATR)`（多空相反）。
- **资金曲线止损**：滚动窗口回撤>阈值暂停策略版本。

**4) 交易成本与滑点**

- 预估冲击成本 `λ * √(notional)`，优化目标加入惩罚项；下单时自适应限价偏移（盘口强度、排队深度）。

**5) 仓位管理细化

- 引入分级杠杆：在波动小且方向明确时放大杠杆，在行情不稳时降低杠杆。

- 使用Kelly 公式修正，让仓位与账户权益的变化挂钩，避免爆仓风险。
---

## 五、策略优胜劣汰与资本分配

**多臂赌博机（Bandit）**

- 回报定义：`R = (PnL_net / risk)`，risk 可用 `VaR`/`vol`/`drawdown`。
- **UCB1**：`score = μ + c * √(ln T / n)`；**Thompson**：对风险调整收益做贝塔/正态后验采样。
- 周期：5min/1h（可配置），末位策略进入 `disabled_until = now()+cooldown`。
- 资本流转有**缓动约束**，避免频繁再平衡引发成本。
- 不要一轮亏损就淘汰策略，可以引入表现平滑期（比如最近 7 天或 50 笔交易）作为考核周期，避免短期波动导致的误判。
---

## 六、Optimizer（参数优化与防过拟合）
参数优化建议引入多目标优化（MOO）
不仅考虑收益最大化，还同时优化回撤最小化、夏普比率最大化。这样不会出现某次优化后，收益高但风险也高得吓人的情况。
**触发条件**：定时 / 业绩劣化 / 市场状态切换。

**流程**

1. 数据切片：滚动窗口 `train -> validate -> forward`（WFO）。
2. 搜索算法：
   - 小维度：拉丁超立方/网格 + 局部爬山（Nelder–Mead）。
   - 中维度：贝叶斯优化（TPE/GP/EI）。
   - 非凸/非平滑：CMA-ES/遗传算法。
3. 目标函数（可切换）：`max Calmar`、`max Sharpe - α*turnover - β*max_dd`、或 `Sortino`。
4. 稳健性检验：
   - **pBO/Deflated Sharpe**、`Combarro 过拟合检验`、`交叉符号鲁棒性`。
   - 参数抖动 ±δ 的性能敏感度（局部平坦优先）。
5. 产出：`best_params`、可信区间、敏感度热图、回测报告（可下载）。
6. 发布：写入 `strategy_versions`，走 canary → 全量。

---

## 七、回测 & 线上一致性

- 事件驱动撮合：重放逐笔/盘口（如 100ms 采样），仿真撮合队列与排队时间。
- 费用模型：费率、资金费、滑点、爆仓规则、强平价模型。
- 时延模型：下单→成交延迟、撤单失败重试。
- 策略沙箱：回测/纸/真三套一致的 `Strategy SDK` 接口，确保线上一致性。

---


---

## 九、Orchestrator / Scheduler（示例计划）

- `*/1 * * * *` 行情健康检查、风控心跳、账户权益快照
- `*/5 * * * *` 策略表现打分（Bandit）、资金再平衡
- `0 */1 * * *` 热门币种扫描（多维打分）
- `10 0 * * *` 日度优化（WFO）
- 事件触发：`业绩劣化/大幅波动/风险阈值` → 立即优化或降杠杆

---

## 十、热门币种打分模型（Hotlist）

得分 `S = w1*VolJump + w2*Turnover + w3*OIΔ + w4*FundingZ + w5*RegimeShift (+ 可选情绪分)`

- `VolJump`：`σ_t / median(σ_{t-7})`
- `Turnover`：`(Vol*Price) / MCap` 或 过去分位
- `OIΔ`：`(OI_t - OI_{t-1}) / OI_{t-1}`
- `FundingZ`：资金费 z 分数
- `RegimeShift`：趋势/震荡切换指示
- 输出 Top-N → `hotlist` 表，需前端人工 `approved=true` 后进入交易白名单。

---

## 十一、API 设计（REST + WS）

**REST（示例）**

- `POST /api/optimizer/run` `{strategy_id, trigger, windows, objective}`
- `POST /api/strategy/{id}/promote` `{version_id, stage}`
- `POST /api/alloc/rebalance` `{mode: bandit|target_vol}`
- `POST /api/risk/limits` `{symbol, max_leverage, max_pos_value}`
- `POST /api/hotlist/approve` `{ts, symbol}`
- `GET /api/metrics/strategy/{id}` → 曲线与 KPI
- `GET /api/audit?entity=orders&from=...`

**WebSocket 频道**

- `ws://.../stream/market/<symbol>` 实时行情
- `ws://.../stream/strategy/<id>` 信号、订单、成交回报
- `ws://.../stream/alerts` 风控/熔断/告警

---

## 十二、前端（React + shadcn/ui）页面

1. **总览看板**：账户权益、昨日/当日 PnL、回撤、在运行策略、风险指示灯（绿/黄/红）。
2. **策略库**：策略卡片（名称、类型、当前版本、阶段、最近绩效）、一键进入回测/优化。
3. **参数优化实验室**：WFO 配置、搜索空间、目标函数、结果表、敏感度热图、对比图。
4. **资金与仓位**：各策略权重、目标 vs 实际、调仓计划、可回滚操作。
5. **风控中心**：限额配置、熔断阈值、止盈止损模板、审批流。
6. **热门币种**：排行榜、评分维度解释、启用开关、白名单管理。
7. **审计与回放**：决策链时间线（信号→风险→下单→成交），可导出 JSON/CSV 报告。
8. 给策略列表加实时 PnL & 风险热力图

9. 仓位列表加多层止盈止损状态指示器

10. 策略优化过程可视化（参数迭代曲线）
---

## 十三、策略切换与灰度（Canary）

- **阶段**：`paper → shadow → canary(<=10%) → prod`。
- 进入 prod 的要求：`年化>阈值`、`MDD<阈值`、`KS 检验通过`、`online slippage within band`。
- 回滚策略：任意风控命中或业绩劣化立即回滚上一个稳定版本。

---

## 十四、告警与熔断

- 账户级：保证金率/资金曲线回撤/风控变量异常。
- 市场级：极端波动、交易所连通/速率异常、撮合延迟异常。
- 策略级：信号失联、下单拒单率上升、滑点 > 阈值、收益偏离回测分布。
- **熔断动作**：降杠杆→减仓→暂停下单→强制平仓（逐级）。

---

## 十五、安全与合规

- API Key 管理：KMS/HashiCorp Vault；最小权限；只读与交易分离。
- 策略与参数审批：RBAC + 审计；生产前双人复核（4-eyes）。
- 日志与留痕：所有自动化决策均落审计表，可回放。

---

## 十六、系统稳定性

建议策略执行与优化分开进程，优化时不会阻塞实盘交易

Redis 用于实时行情缓存 + 队列处理，数据库存历史与回测结果

API 增加速率限制，防止策略短时间内请求过多导致交易所限流