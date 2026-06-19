# Phase 10: ML 预测建模实战化 — Design Spec

> **Status**: Draft
> **Date**: 2026-06-18
> **Depends on**: Phase 1-9 (完整基础设施)

## Motivation

Phase 3 建立了 Python gRPC Sidecar，Phase 4 建立了 AI Agent 系统。但 Python 侧的 `ml/` 目录完全是空壳——`engine.py` 中 `TrainModel`、`Predict`、`ListModels` 三个方法全部返回 `"not implemented"`。Go 侧没有 ML 客户端、没有模型注册表、没有 ML 工作流节点。

这意味着 QuantFlow 当前可以做**描述性分析**（面板看行情、回测验证策略、手动写因子），但完全没有**预测能力**——无法回答"明天这个因子会怎么样？""历史数据里还能挖出什么 Alpha？""AI 能不能自己学交易？"

Phase 10 将 ML 从存根变为核心生产力。通过四层能力（收益预测→因子挖掘→强化学习交易→风险建模），把 Python 生态中成熟的 ML 工具链（XGBoost/LightGBM/PyTorch/gplearn/Gym）与 Go 的工作流编排体系深度融合。

### 当前状态 vs 目标

| 能力 | 当前（Phase 9） | 目标（Phase 10） |
|------|---------------|-----------------|
| 模型训练 | ml/engine.py 存根，返回 "not implemented" | TreeEngine + DeepEngine 完整训练管线 |
| 模型预测 | 无 | PredictNode 接入工作流，输出可直接回测验证 |
| 因子挖掘 | 手动从 25 因子库选择和组合 | 遗传规划自动发现新因子，IC/IR 评估 |
| 强化学习 | 无 | Gym 环境 + PPO/DQN/SAC，RL 可直接输出交易信号 |
| 风险建模 | RiskMetrics 用历史数据算 VaR（Go 侧） | Python GARCH 族 + 协方差矩阵估计 |
| 模型管理 | 无 | SQLite 模型注册表 + 文件系统双轨 |
| ML 前端 | 无 | 4 个面板：模型管理、预测仪表盘、因子挖掘、RL 监控 |

### 优先级排序

用户明确：**收益预测 → 因子挖掘 → 强化学习交易 → 风险建模**

## Design

### 1. 架构全景

```
Frontend (Vue 3)
┌──────────────────────────────────────────────────────────────────┐
│  ModelRegistry    PredictionDashboard  AlphaMiningWorkspace  RLMonitor │
│  模型 CRUD+搜索   6 视图预测分析        GP 配置+结果浏览      实时训练曲线  │
│  拖入→生成节点    [+]→生成 PredictNode  [注册]→新建因子       [保存]→模型入库 │
└──────────────────────────┬───────────────────────────────────────┘
                           │ Wails IPC
                           │
Go Backend
┌──────────────────────────┼───────────────────────────────────────┐
│                   mlStore (Pinia)                                 │
│  ┌───────────────────────┼────────────────────────────────────┐  │
│  │              Workflow Engine                                │  │
│  │  FeatureEngineer → TrainModel → Predict → EvaluateModel    │  │
│  │  AlphaMining → EvaluateModel → rank_select → StrategyNode  │  │
│  │  RLEnv → RLTrain → RLPredict → PlaceOrder                  │  │
│  │  RiskModel → RiskMetrics                                    │  │
│  └───────────────────────┬────────────────────────────────────┘  │
│                          │                                        │
│  ┌───────────────────────┴────────────────────────────────────┐  │
│  │                    ML Domain Layer                           │  │
│  │  ModelRegistry  │  ModelManager  │  Evaluator  │  Types     │  │
│  └───────────────────────┬────────────────────────────────────┘  │
│                          │                                        │
│  ┌───────────────────────┴────────────────────────────────────┐  │
│  │              python/ml_client.go (gRPC Client)               │  │
│  │  Train | Predict | Evaluate | AlphaMining | RLTrain | RLPredict│
│  └───────────────────────┬────────────────────────────────────┘  │
│                          │ gRPC (localhost:50051)                 │
│                          │ Arrow IPC for DataFrames              │
│                          │ RLTrain: server-streaming progress     │
└──────────────────────────┼──────────────────────────────────────┘
                           │
Python gRPC Sidecar
┌──────────────────────────┼──────────────────────────────────────┐
│                    MLService (engine.py)                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐  ┌────────────┐  │
│  │TreeEngine│  │DeepEngine│  │AlphaMiningEng│  │  RLEngine  │  │
│  │ XGBoost  │  │ LSTM     │  │ genetic.py   │  │ env.py     │  │
│  │ LightGBM │  │ Transfrm │  │ deep_search  │  │ ppo/dqn/sac│  │
│  └──────────┘  └──────────┘  │ evaluator.py │  └────────────┘  │
│                               └──────────────┘                  │
│  ┌──────────┐  ┌──────────────┐                                 │
│  │RiskEngine│  │ serialization│  model files: .joblib / .pt     │
│  │ GARCH    │  │ registry.py  │                                 │
│  └──────────┘  └──────────────┘                                 │
└──────────────────────────────────────────────────────────────────┘
```

### 2. Python 端设计

#### 2.1 包结构

```
python/src/ml/
├── __init__.py
├── engine.py              # 重构：MLService gRPC 入口，路由到各子引擎
├── tree_engine.py          # NEW: XGBoost/LightGBM 训练+预测+特征重要性
├── deep_engine.py          # NEW: PyTorch LSTM/Transformer 时序预测
├── alpha_mining/           # NEW: 因子挖掘子系统
│   ├── __init__.py
│   ├── genetic.py          #   遗传规划 (gplearn)，符号回归搜索因子公式
│   ├── deep_search.py      #   深度因子搜索（AutoEncoder 特征组合）
│   └── evaluator.py        #   因子评估：IC/IR/分层回测
├── rl/                     # NEW: 强化学习子系统
│   ├── __init__.py
│   ├── env.py              #   Gym 交易环境（OHLCV → state，action → reward）
│   ├── algorithms/
│   │   ├── __init__.py
│   │   ├── ppo.py          #   PPO 算法实现
│   │   ├── dqn.py          #   DQN 算法实现
│   │   └── sac.py          #   SAC 算法实现
│   └── replay.py           #   经验回放缓冲区
├── risk/                   # NEW: 风险建模
│   ├── __init__.py
│   ├── garch.py            #   GARCH/GJR-GARCH/EGARCH 波动率建模
│   └── covariance.py       #   Ledoit-Wolf / DCC-GARCH 协方差估计
├── registry.py             # NEW: 内存模型缓存（主注册表在 Go SQLite）
└── serialization.py        # NEW: 统一序列化接口（joblib + torch.save）
```

#### 2.2 统一接口约定

所有 Engine 遵循相同模式：

```python
class BaseEngine:
    def train(self, features: pa.Table, targets: pa.Table, params: dict) -> dict:
        """训练模型，返回 {model_id, metrics, file_path}"""
        ...

    def predict(self, model_id: str, features: pa.Table) -> pa.Array:
        """加载模型执行推理，返回 Arrow 数组"""
        ...

    def evaluate(self, model_id: str, features: pa.Table, targets: pa.Table) -> dict:
        """评估模型，返回 {metric_name: value, ...}"""
        ...

    def feature_importance(self, model_id: str) -> dict:
        """返回特征重要性排序（TreeEngine 原生支持，DeepEngine 用 permutation）"""
        ...
```

#### 2.3 TreeEngine（收益预测引擎）

- **模型类型**：`xgboost.XGBRegressor`、`xgboost.XGBClassifier`、`lightgbm.LGBMRegressor`、`lightgbm.LGBMClassifier`
- **目标类型**：回归（预测未来 N 日收益率）、分类（预测涨/跌/平方向）
- **核心功能**：训练、预测、特征重要性、早停（early stopping）、交叉验证
- **序列化**：`joblib.dump()` → 文件；超参+指标 → Go SQLite

#### 2.4 DeepEngine（深度时序引擎）

- **模型类型**：LSTM、Transformer（基于 PyTorch）
- **输入格式**：(batch, seq_len, features) 三维张量
- **核心功能**：滑动窗口数据制备、训练/验证集划分、早停、GPU 检测
- **序列化**：`torch.save()` → .pt 文件
- **注意**：DeepEngine 是可选组件（torch 安装较大），如果 torch 未安装，gRPC 返回明确错误

#### 2.5 AlphaMiningEngine（因子挖掘引擎）

- **遗传规划**：基于 gplearn，从 25 基础因子池自动组合演化
  - 个体表示：因子公式 AST 树
  - 适应度函数：IC 绝对值、IR、或组合指标
  - 操作符：+, -, *, /, rank, ts_sum, ts_mean, ts_std, cross_over, scale
  - 停止条件：达到目标代数 / IC 收敛
- **深度搜索**（可选）：AutoEncoder 降维 + K-Means 聚类发现因子组
- **评估管线**：每个候选因子 → 计算 IC 序列 → 分层回测 → 输出 IC/IR/Sharpe
- **输出**：因子公式（可注册为新的 FactorNode 直接在工作流中使用）

#### 2.6 RLEngine（强化学习引擎）

- **交易环境（Gym）**：
  - State: [持仓, 现金比例, 最近 N 根 K 线的 OHLCV, 技术指标]
  - Action Space: Discrete(3) = {sell(-1), hold(0), buy(1)} 或 Continuous(1) = 仓位比例
  - Reward: 夏普比率增量 / 对数收益率
  - 终止条件：最大 step / 强平
- **算法**：PPO（主力）、DQN（离散动作）、SAC（连续动作）
- **流式训练**：`RLTrain` RPC 为 server-streaming，每 episode 推送 `{episode, reward, sharpe, epsilon}`
- **推理**：加载训练好的 policy network，输入 observation 输出 action distribution

#### 2.7 RiskEngine（风险建模引擎）

- **GARCH 族**：GARCH(1,1)、GJR-GARCH（非对称）、EGARCH
- **协方差矩阵**：Sample Cov、Ledoit-Wolf Shrinkage、DCC-GARCH
- **输出**：条件波动率序列、协方差矩阵 → 送入 Go 侧 RiskMetrics 节点

#### 2.8 依赖新增

```toml
# pyproject.toml 新增
"xgboost>=2.0",
"lightgbm>=4.0",
"scikit-learn>=1.3",
"torch>=2.1",           # DeepEngine + RLEngine（可选）
"gplearn>=0.4",         # AlphaMining
"gymnasium>=0.29",      # RL env
"arch>=6.0",            # GARCH
```

### 3. Go 端设计

#### 3.1 新增包结构

```
internal/ml/              # NEW: ML 领域包
├── registry.go           #   ModelRegistry — 模型 CRUD、状态管理
├── manager.go            #   ModelManager — 训练/预测任务调度与生命周期
├── evaluator.go          #   模型评估器 — IC/IR/Sharpe/MSE 计算
└── types.go              #   共享类型定义

internal/python/
└── ml_client.go          # NEW: Python ML gRPC 客户端

internal/workflow/nodes/
├── feature_engineer.go   # NEW: FeatureEngineerNode — 特征标准化/缺失值/滞后对齐
├── train_model.go        # NEW: TrainModelNode — 触发 Python 训练
├── predict.go            # NEW: PredictNode — 加载模型执行推理
├── evaluate_model.go     # NEW: EvaluateModelNode — 模型评估
├── alpha_mining.go       # NEW: AlphaMiningNode — 因子挖掘
├── rl_env.go             # NEW: RLEnvNode — 构建 Gym 环境
├── rl_train.go           # NEW: RLTrainNode — RL 训练（接收流式进度）
├── rl_predict.go         # NEW: RLPredictNode — RL 推理
└── risk_model.go          # NEW: RiskModelNode — 风险建模

internal/storage/migrations/
└── 010_ml_models.go      # NEW: 模型/预测/评估表
```

#### 3.2 核心类型

```go
// internal/ml/types.go

type ModelType string
const (
    ModelTypeXGBoost    ModelType = "xgboost"
    ModelTypeLightGBM   ModelType = "lightgbm"
    ModelTypeLSTM       ModelType = "lstm"
    ModelTypeTransformer ModelType = "transformer"
    ModelTypePPO        ModelType = "ppo"
    ModelTypeDQN        ModelType = "dqn"
    ModelTypeSAC        ModelType = "sac"
    ModelTypeGARCH      ModelType = "garch"
)

type ModelCategory string
const (
    CategoryPrediction   ModelCategory = "prediction"
    CategoryAlphaMining  ModelCategory = "alpha_mining"
    CategoryRL           ModelCategory = "rl"
    CategoryRisk         ModelCategory = "risk"
)

type ModelStatus string
const (
    ModelStatusTraining  ModelStatus = "training"
    ModelStatusReady     ModelStatus = "ready"
    ModelStatusFailed    ModelStatus = "failed"
    ModelStatusArchived  ModelStatus = "archived"
)

type MLModel struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    ModelType   ModelType         `json:"model_type"`
    Category    ModelCategory     `json:"category"`
    Hyperparams map[string]string `json:"hyperparams"`
    Metrics     map[string]float64 `json:"metrics"`
    FilePath    string            `json:"file_path"`
    FileBytes   []byte            `json:"-"`           // 小模型直接存 blob
    Status      ModelStatus       `json:"status"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}
```

#### 3.3 ModelRegistry

```go
// internal/ml/registry.go

type ModelRegistry struct {
    db *sql.DB
}

// CRUD + 状态机
func (r *ModelRegistry) Create(ctx context.Context, model *MLModel) error
func (r *ModelRegistry) Get(ctx context.Context, id string) (*MLModel, error)
func (r *ModelRegistry) List(ctx context.Context, filter ModelFilter) ([]*MLModel, error)
func (r *ModelRegistry) UpdateStatus(ctx context.Context, id string, status ModelStatus) error
func (r *ModelRegistry) Archive(ctx context.Context, id string) error
func (r *ModelRegistry) Delete(ctx context.Context, id string) error

// 存储决策
func (r *ModelRegistry) SaveModelFile(model *MLModel) error   // >1MB → 文件系统; ≤1MB → SQLite blob
func (r *ModelRegistry) LoadModelFile(id string) ([]byte, error)
```

状态机：`training → ready | failed`，`ready → archived`，`archived → (可恢复为 ready)`

#### 3.4 新增工作流节点（9 个，类别：ML）

| 节点 | 输入端口 | 输出端口 | 核心逻辑 |
|------|---------|---------|---------|
| **FeatureEngineer** | `ohlcv_data`, `factors` | `feature_matrix` | 标准化、缺失值填充、滞后对齐、防 look-ahead bias |
| **TrainModel** | `feature_matrix`, `target` | `model_id`, `train_metrics` | 调用 Python Bridge 训练，注册到 ModelRegistry |
| **Predict** | `model_id`, `feature_matrix` | `predictions` | 从 Registry 加载模型，调用 Python 推理 |
| **EvaluateModel** | `predictions`, `actual` | `evaluation_report` | IC/IR/MSE/Sharpe/R²，写入 ml_evaluations 表 |
| **AlphaMining** | `factor_pool`, `ohlcv_data` | `new_factors`, `factor_scores` | 调用 Python 遗传规划，返回因子公式+IC |
| **RLEnv** | `ohlcv_data`, `factors` | `env_config` | 配置 Gym 环境：state 维度、reward 函数、终止条件 |
| **RLTrain** | `env_config`, `algorithm` | `model_id`, `reward_curve` | 调用 Python RL 训练（stream），实时输出 reward |
| **RLPredict** | `model_id`, `observation` | `action` | 加载 policy network，输出交易动作 |
| **RiskModel** | `returns_data` | `volatility`, `covariance` | 调用 Python RiskEngine，输出波动率序列+协方差矩阵 |

> **注**：FeatureEngineer + TrainModel + Predict + EvaluateModel + AlphaMining + RLEnv + RLTrain + RLPredict + RiskModel = **9 个节点**（非 8 个）

#### 3.5 关键数据流：收益预测闭环

```
DataLoader → Factor → FeatureEngineer → TrainModel → Predict → EvaluateModel → BacktestNode
                                                                                    ↓
                    [训练期数据]              [样本外预测]     [模型评估]      [策略回测验证]
```

关键约束：
- **防 look-ahead bias**：FeatureEngineer 确保 t 时刻特征只用 ≤t 时刻数据
- **样本内外分离**：TrainModel 在训练集上训练，EvaluateModel/BacktestNode 在验证集/测试集上评估
- **模型版本化**：每次 TrainModel 生成新 model_id，旧模型默认归档（可恢复）

#### 3.6 gRPC Proto 扩展

```protobuf
// ml.proto 重构

service MLService {
  // 训练
  rpc Train(TrainRequest) returns (TrainResponse);
  // 预测
  rpc Predict(PredictRequest) returns (PredictResponse);
  // 评估
  rpc Evaluate(EvaluateRequest) returns (EvaluateResponse);
  // 因子挖掘
  rpc AlphaMining(AlphaMiningRequest) returns (AlphaMiningResponse);
  // RL 训练（server-streaming 进度）
  rpc RLTrain(RLTrainRequest) returns (stream RLTrainUpdate);
  // RL 推理
  rpc RLPredict(RLPredictRequest) returns (RLPredictResponse);
  // 风险建模
  rpc RiskModel(RiskModelRequest) returns (RiskModelResponse);
}

message TrainRequest {
  string model_type = 1;        // xgboost/lightgbm/lstm/transformer
  bytes features = 2;           // Arrow IPC 编码的 DataFrame
  bytes targets = 3;            // Arrow IPC 编码的目标变量
  map<string, string> hyperparams = 4;
  string target_type = 5;       // regression/classification
  int32 forecast_horizon = 6;   // 预测周期（1/5/20 日）
}

message TrainResponse {
  string model_id = 1;
  bytes model_bytes = 2;        // 序列化的模型文件（大模型通过文件路径引用）
  string model_file_path = 3;   // 模型文件路径（备选）
  map<string, double> metrics = 4;  // train_rmse, val_rmse, train_mae, ...
  int64 train_time_ms = 5;
  repeated FeatureImportance feature_importance = 6;
}

message FeatureImportance {
  string feature_name = 1;
  double importance = 2;
}

message PredictRequest {
  string model_id = 1;
  bytes features = 2;           // Arrow IPC
}

message PredictResponse {
  repeated double predictions = 1;
  int64 predict_time_ms = 2;
}

message EvaluateRequest {
  string model_id = 1;
  bytes features = 2;
  bytes actuals = 3;
}

message EvaluateResponse {
  map<string, double> metrics = 1;  // ic, ir, mse, mae, r2, sharpe
  int64 evaluate_time_ms = 2;
}

message AlphaMiningRequest {
  repeated string base_factor_names = 1;  // 基础因子池
  bytes factor_data = 2;                  // Arrow IPC 编码的因子矩阵
  bytes returns_data = 3;                 // Arrow IPC 编码的未来收益
  int32 population_size = 4;              // 种群大小（默认 200）
  int32 generations = 5;                  // 代数（默认 50）
  double crossover_rate = 6;              // 交叉率（默认 0.7）
  double mutation_rate = 7;               // 变异率（默认 0.1）
  string fitness_metric = 8;              // ic/ir/sharpe/composite
  int32 top_k = 9;                        // 返回前 K 个因子（默认 20）
}

message AlphaMiningResponse {
  repeated DiscoveredFactor factors = 1;
  int64 mining_time_ms = 2;
}

message DiscoveredFactor {
  string formula = 1;           // 可读的因子公式
  double ic = 2;
  double ir = 3;
  double sharpe = 4;            // 分层回测夏普
}

message RLTrainRequest {
  string algorithm = 1;          // ppo/dqn/sac
  map<string, string> hyperparams = 2;
  bytes ohlcv_data = 3;          // Arrow IPC
  int32 total_episodes = 4;
  int32 episode_length = 5;      // 每 episode 最大步数
  string action_space = 6;       // discrete/continuous
}

message RLTrainUpdate {
  int32 episode = 1;
  double reward = 2;
  double sharpe = 3;
  double epsilon = 4;            // 探索率（PPO 用）
  int32 steps = 5;
}

message RLPredictRequest {
  string model_id = 1;
  bytes observation = 2;         // 当前状态向量
}

message RLPredictResponse {
  int32 action = 1;              // 离散动作: -1/0/1
  double action_value = 2;       // 连续动作: 仓位比例
  map<string, double> action_probs = 3;  // 动作概率分布
}

message RiskModelRequest {
  string model_type = 1;         // garch/gjr_garch/egarch/dcc/covariance
  bytes returns_data = 2;        // Arrow IPC
  map<string, string> params = 3;
}

message RiskModelResponse {
  bytes result_data = 1;         // Arrow IPC: 波动率序列/协方差矩阵
  map<string, double> metrics = 2;
  int64 compute_time_ms = 3;
}
```

#### 3.7 Python ML Client

```go
// internal/python/ml_client.go

type MLClient struct {
    conn   *grpc.ClientConn
    client proto.MLServiceClient
}

func (c *MLClient) Train(ctx context.Context, req *proto.TrainRequest) (*proto.TrainResponse, error)
func (c *MLClient) Predict(ctx context.Context, req *proto.PredictRequest) (*proto.PredictResponse, error)
func (c *MLClient) Evaluate(ctx context.Context, req *proto.EvaluateRequest) (*proto.EvaluateResponse, error)
func (c *MLClient) AlphaMining(ctx context.Context, req *proto.AlphaMiningRequest) (*proto.AlphaMiningResponse, error)
func (c *MLClient) RLTrain(ctx context.Context, req *proto.RLTrainRequest) (<-chan *proto.RLTrainUpdate, error)  // 返回 channel
func (c *MLClient) RLPredict(ctx context.Context, req *proto.RLPredictRequest) (*proto.RLPredictResponse, error)
func (c *MLClient) RiskModel(ctx context.Context, req *proto.RiskModelRequest) (*proto.RiskModelResponse, error)
```

`RLTrain` 返回 Go channel，每收到一个 stream message 就推入 channel，前端通过 Wails SSE 实时消费。

### 4. 存储设计（Migration 010）

```sql
-- 模型注册表
CREATE TABLE IF NOT EXISTS ml_models (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    model_type  TEXT NOT NULL,   -- xgboost/lightgbm/lstm/transformer/ppo/dqn/sac/garch
    category    TEXT NOT NULL,   -- prediction/alpha_mining/rl/risk
    hyperparams TEXT DEFAULT '{}',    -- JSON map
    metrics     TEXT DEFAULT '{}',    -- JSON: {"sharpe": 1.82, "ic": 0.08, ...}
    file_path   TEXT,            -- 模型文件绝对路径（>1MB 时使用）
    file_bytes  BLOB,            -- 模型二进制（≤1MB 时使用）
    status      TEXT DEFAULT 'training',  -- training/ready/failed/archived
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_ml_models_type ON ml_models(model_type);
CREATE INDEX IF NOT EXISTS idx_ml_models_category ON ml_models(category);
CREATE INDEX IF NOT EXISTS idx_ml_models_status ON ml_models(status);

-- 预测记录
CREATE TABLE IF NOT EXISTS ml_predictions (
    id          TEXT PRIMARY KEY,
    model_id    TEXT NOT NULL REFERENCES ml_models(id) ON DELETE CASCADE,
    symbol      TEXT NOT NULL,
    date        TEXT NOT NULL,
    prediction  REAL NOT NULL,
    actual      REAL,            -- 事后填充
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_ml_predictions_model ON ml_predictions(model_id);
CREATE INDEX IF NOT EXISTS idx_ml_predictions_symbol_date ON ml_predictions(symbol, date);

-- 评估记录
CREATE TABLE IF NOT EXISTS ml_evaluations (
    id          TEXT PRIMARY KEY,
    model_id    TEXT NOT NULL REFERENCES ml_models(id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL,   -- ic/ir/sharpe/mse/mae/accuracy/precision/recall/r2
    value       REAL NOT NULL,
    period      TEXT NOT NULL,   -- train/validation/test
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_ml_evaluations_model ON ml_evaluations(model_id);
```

### 5. 前端设计

#### 5.1 新增 Pinia Store：`mlStore`

```typescript
// stores/mlStore.ts
interface MLStoreState {
  models: MLModel[]
  selectedModel: MLModel | null
  trainingJobs: TrainingJob[]
  trainingProgress: Record<string, number>      // model_id → progress%
  predictions: Prediction[]
  miningJobs: AlphaMiningJob[]
  discoveredFactors: DiscoveredFactor[]
  rlTrainingCurves: Record<string, number[]>    // model_id → reward history
  rlActions: RLAction[]
}
```

#### 5.2 面板 1：ModelRegistry（模型管理面板）

- **功能**：模型列表（表格，支持按类型/类别/状态过滤和名称搜索）、模型详情展开（超参数 JSON、评估指标、特征重要性柱状图）、训练新模型入口（弹窗表单：选择模型类型→配置超参数→选择数据源）、模型操作（归档/删除/导出）、拖入工作流生成 PredictNode
- **图标**：🧠 / 快捷键：Ctrl+M

#### 5.3 面板 2：PredictionDashboard（预测仪表盘）

- **功能**：顶部模型+品种选择器、6 个 ECharts 视图（预测收益分布直方图、IC 时序折线图、分行业 IC 柱状图、预测 vs 实际散点图、分位数分层回测曲线、近期预测明细表）、[+] 生成 PredictNode 按钮
- **图标**：📈 / 快捷键：Ctrl+Shift+P

#### 5.4 面板 3：AlphaMiningWorkspace（因子挖掘工作区）

- **功能**：基础因子池多选（从 25 因子库 + 已注册自定义因子中选择）、GP 参数配置表单（种群/代数/选择方式/交叉率/变异率/适应度函数）、运行按钮+进度条、结果列表（因子公式+IC+IR+Sharpe，按 IC 降序）、每个结果有 [注册为因子] 和 [回测验证] 按钮
- **图标**：⛏️ / 快捷键：Ctrl+Shift+A

#### 5.5 面板 4：RLMonitor（RL 训练监控）

- **功能**：环境+算法选择器、4 个 ECharts 视图（实时 Reward Curve 折线图——通过 SSE 流式更新、Drawdown 面积图、Action 分布饼图、Position Size 直方图）、训练控制按钮（启动/暂停/保存模型）、最新 episode 指标展示
- **图标**：🤖 / 快捷键：Ctrl+Shift+R

#### 5.6 面板注册

4 个新面板注册到 Terminal Mode → 面板菜单 → 「ML / AI」分类。

### 6. 子阶段划分

| 子阶段 | 内容 | 依赖 | 预计产出 |
|--------|------|------|---------|
| **10.1 收益预测** | TreeEngine + DeepEngine + FeatureEngineer + TrainModel + Predict + EvaluateModel 节点 + ModelRegistry 面板 + PredictionDashboard 面板 | Phase 3 (gRPC), Phase 9 (因子节点) | 端到端预测→回测闭环 |
| **10.2 因子挖掘** | AlphaMiningEngine + AlphaMiningNode + AlphaMiningWorkspace 面板 | 10.1 (模型注册+评估基础设施) | 自动发现新因子并注册 |
| **10.3 强化学习** | RLEngine + RLEnv/Train/Predict 节点 + RLMonitor 面板 | 10.1 (DeepEngine 共用 torch) | RL 训练→实盘/模拟交易 |
| **10.4 风险建模** | RiskEngine + RiskModelNode + RiskDashboard 面板扩展 | 10.1 (基础管线) | GARCH 波动率+协方差矩阵 |

### 7. 数据流详解

#### 7.1 收益预测闭环
```
DataLoader(t-365:t) → Factor(25α) → FeatureEngineer(lag对齐,标准化)
  → TrainModel(训练集: t-365:t-60) → model_id + metrics
  → Predict(测试集: t-60:t) → predictions
  → EvaluateModel(predictions vs actual) → IC/IR/Sharpe
  → BacktestNode(用 predictions 构建策略信号) → 权益曲线+回测指标
```

#### 7.2 因子挖掘闭环
```
AlphaMining(因子池, OHLCV, GP配置) → 候选因子公式列表
  → [人工筛选/注册] → FactorNode(新因子)
  → FeatureEngineer → Predict → EvaluateModel → BacktestNode
  → 验证成功 → 因子入库（下次挖掘时可作为基础因子）
```

#### 7.3 RL 交易闭环
```
DataLoader → RLEnv(构建 Gym 状态) → RLTrain(PPO/DQN, stream reward)
  → model_id → RLPredict(实时 observation → action)
  → PlaceOrder(action → 实际订单) → 组合监控
```

#### 7.4 风险建模闭环
```
DataLoader(returns) → RiskModel(GARCH/协方差) → 波动率序列/协方差矩阵
  → RiskMetrics(VaR/CVaR 计算) → RiskDashboard(仪表盘更新)
```

### 8. 测试策略

| 层 | 测试类型 | 覆盖目标 |
|----|---------|---------|
| Python TreeEngine | 单元测试 | 训练/预测/评估/序列化/特征重要性 |
| Python DeepEngine | 单元测试 | 数据制备/LSTM/Transformer/早停/GPU 回退 |
| Python AlphaMining | 单元测试 | 遗传规划收敛/因子公式合法性/IC 计算 |
| Python RLEngine | 单元测试 | Gym 环境/DQN/PPO/SAC 收敛 |
| Python RiskEngine | 单元测试 | GARCH 拟合/协方差估计 |
| Python gRPC | 集成测试 | MLService 各 RPC + Arrow IPC 往返 |
| Go ml_client | 单元测试 | gRPC 调用 + stream 消费 + 错误处理 |
| Go ModelRegistry | 单元测试 | CRUD + 状态机 + 文件/blob 双轨 |
| Go 工作流节点 | 单元测试 | 9 个节点的输入验证+输出正确性 |
| Go DAG 集成 | 集成测试 | DataLoader→Factor→FeatureEngineer→TrainModel 全链路 |
| Frontend | 组件测试 | 4 个面板的渲染+交互 |

### 9. 风险和缓解

| 风险 | 缓解措施 |
|------|---------|
| **torch 依赖过重**（~2GB） | torch 标记为可选依赖；DeepEngine/RLEngine 优雅降级，未安装时返回明确错误 |
| **训练耗时长阻塞工作流** | TrainModel 节点支持 timeout 参数；长训练用异步模式（节点返回 training job_id，后续轮询） |
| **look-ahead bias** | FeatureEngineer 严格执行时序对齐；测试中包含 bias 检测用例 |
| **gplearn 收敛慢/过拟合** | 限制代数上限；验证集 IC 作为早停标准；因子公式长度限制 |
| **RL 训练不稳定** | 多次随机种子训练取最佳；reward curve 实时可见便于人工中止 |
| **模型文件大小** | 1MB 阈值自动切换 blob↔文件；大模型提示用户磁盘空间 |

## Acceptance Criteria

### 10.1 收益预测
- [ ] TreeEngine 支持 XGBoost/LightGBM 训练/预测/评估/特征重要性
- [ ] DeepEngine 支持 LSTM/Transformer 时序预测（torch 可用时）
- [ ] FeatureEngineer 节点能标准化、填充缺失值、滞后对齐、防 look-ahead bias
- [ ] TrainModel 节点：接收特征矩阵+目标 → 调用 Python 训练 → 返回 model_id
- [ ] Predict 节点：接收 model_id+特征 → 返回预测值
- [ ] EvaluateModel 节点：接收预测值+实际值 → 输出 IC/IR/MSE/Sharpe
- [ ] 端到端工作流：DataLoader→Factor→FeatureEngineer→TrainModel→Predict→EvaluateModel→BacktestNode 可执行
- [ ] ModelRegistry 面板：模型 CRUD、过滤、搜索、拖入工作流
- [ ] PredictionDashboard 面板：6 视图展示预测结果

### 10.2 因子挖掘
- [ ] AlphaMiningEngine 遗传规划能从基础因子池自动发现新因子
- [ ] AlphaMiningNode 工作流节点可配置 GP 参数并执行
- [ ] AlphaMiningWorkspace 面板：因子池选择、GP 配置、结果浏览、注册
- [ ] 发现的因子可注册为 FactorNode 供后续工作流使用

### 10.3 强化学习
- [ ] RLEngine 提供标准 Gym 交易环境
- [ ] PPO/DQN/SAC 三种算法可训练并收敛
- [ ] RLTrain RPC 流式返回每 episode 训练进度
- [ ] RLTrainNode 接收环境配置+算法 → 返回 model_id+reward_curve
- [ ] RLPredictNode 接收 observation → 输出 action
- [ ] RLMonitor 面板：实时 reward curve + 训练控制

### 10.4 风险建模
- [ ] RiskEngine 支持 GARCH/GJR-GARCH/EGARCH 波动率建模
- [ ] RiskEngine 支持 Ledoit-Wolf/DCC-GARCH 协方差估计
- [ ] RiskModelNode 输出波动率序列/协方差矩阵
- [ ] RiskDashboard 面板扩展显示条件波动率+协方差热力图

### 横切
- [ ] Migration 010 正确创建 ml_models/ml_predictions/ml_evaluations 表
- [ ] mlStore 正确管理所有 ML 状态
- [ ] 4 个面板正确注册到 Terminal Mode
- [ ] 9 个工作流节点注册到 NodeRegistry
- [ ] 所有 Python 测试通过（pytest）
- [ ] 所有 Go 测试通过（go test）
- [ ] 所有前端测试通过（vitest）
- [ ] CHANGELOG 更新
- [ ] 版本日期更新

## Risks / Trade-offs

1. **复杂度风险**：Phase 10 跨越 4 个 ML 子领域，每个都足够做一个独立 Phase。缓解：10.1 做完就停手评估，确认方向正确再继续 10.2。
2. **依赖风险**：torch 和 gplearn 是重量级依赖。缓解：标记为可选，非破坏性降级。
3. **模型漂移**：模型训练完到部署期间市场可能变化。缓解：EvaluateModel 持续监控 IC 衰减；面板显示 IC 时序便于判断模型是否失效。
4. **训练资源**：用户桌面跑 PyTorch 训练可能很慢。缓解：小模型优先（XGBoost 秒级训练）；RL 支持 episode 上限；未来可选云端训练。
5. **与现有因子/回测系统的边界**：AlphaMining 发现的是"公式型因子"还是"ML 模型型因子"？设计选择：公式型——可以直接注册为 FactorNode 在工作流中使用，透明可解释。
