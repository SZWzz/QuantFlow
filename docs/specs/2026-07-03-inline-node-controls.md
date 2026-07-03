# Inline Node Controls — 卡片内联控件

## Motivation

当前 QuantFlow Workflow 模式的节点卡片参数编辑能力较弱：所有参数渲染为纯文本 `key: value`，点击后出现 inline text input，无类型感知控件。用户需要打开右侧 PropertyPanel 才能获得稍好的编辑体验。ComfyUI 的做法是：所有 primitive 类型的参数（INT/FLOAT/STRING/BOOLEAN/COMBO）直接在卡片上渲染为交互式控件（slider、dropdown、toggle、text input），非 primitive 类型（MODEL/LATENT/IMAGE 等）渲染为连接端口。当端口有连线时，对应的 widget 自动隐藏。

**目标**：将 ComfyUI 的卡片内联控件模式引入 QuantFlow，让所有节点参数在卡片上直接可交互编辑，PropertyPanel 退化为只显示错误策略、状态信息等元数据。

## Design

### 数据流

```
Go ParamDef (schema 扩展)
  ↓ ListNodes() / GetNodeSchema()
NodeMeta (含增强 param schema)
  ↓ Wails IPC
Frontend nodeMetaCache
  ↓
CustomNode.vue 渲染类型感知控件
  ↓ updateParams → workflow store → node.data.params
```

### Go 端变更 — ParamDef 扩展

当前 `ParamDef` 缺乏控件所需的元信息。扩展为：

```go
// ParamDef describes a configurable parameter for a node.
type ParamDef struct {
    Name        string   `json:"name"`
    Type        string   `json:"type"`    // "int", "float", "string", "bool", "select"
    Default     any      `json:"default,omitempty"`
    Description string   `json:"description,omitempty"`
    Required    bool     `json:"required"`
    // 以下为控件增强字段
    Min         *float64 `json:"min,omitempty"`    // INT/FLOAT 最小值
    Max         *float64 `json:"max,omitempty"`    // INT/FLOAT 最大值
    Step        *float64 `json:"step,omitempty"`   // INT/FLOAT 步进
    Options     []string `json:"options,omitempty"` // select 类型选项列表
    // portForParam 命名约定：若存在同名的 input port，widget 在端口连线时隐藏
    // 该映射关系在 node.InputPorts() 定义
}
```

同时新增 `string_array` → `select` 类型重命名，因为 select 更能表达 UI 含义。

所有现有节点需要补全 Min/Max/Step/Options 字段。

### 控件类型映射规则

| ParamDef.Type | 控件 | 隐藏条件 |
|---|---|---|
| `int` | 数字输入 + 步进按钮 (min/max/step) | 同名 port 有连线 |
| `float` | 数字输入 + 步进按钮 (min/max/step/round) | 同名 port 有连线 |
| `string` | 单行文本输入 | 同名 port 有连线 |
| `bool` | Toggle 开关 | 同名 port 有连线 |
| `select` | 下拉选择框 (Options) | 同名 port 有连线 |

### 连线/控件互斥逻辑

当一个 param 与某个 input port **同名**时，它们共享同一个输入源：
- **port 无连线** → 卡片显示 widget，用户直接设置值
- **port 有连线** → 卡片隐藏 widget，值从上游节点来
- **端口连线后被删除** → widget 重新出现

实现方式：在 CustomNode.vue 中检查每个 param 名称是否匹配某个 input port，若匹配且该 port 有连线，则隐藏 widget。

### PropertyPanel 简化

当前 PropertyPanel 渲染了 `visibleParams`，与卡片功能重叠。简化后只保留：
- 错误策略 (stop/skip/retry)
- 节点 ID / Type
- 执行状态 / 错误信息
- Pin to Terminal 按钮
- Input/Output 端口列表（仅查看）

参数编辑完全移至卡片上。

## 修改文件清单

### Go 后端

| 文件 | 改动 |
|---|---|
| `internal/workflow/node.go` | 扩展 `ParamDef` 结构体，添加 Min/Max/Step/Options/Required |
| `internal/workflow/registry.go` | 不用改 (ListAll 直接序列化 ParamDef) |
| `app.go` | 新增 `GetNodeSchema(nodeType)` API（可选完整 schema） |
| `internal/workflow/nodes/sma.go` | 补全 ParamSchema: period → `{Min:1, Max:9999}` |
| `internal/workflow/nodes/bollinger.go` | 补全 ParamSchema: period/multiplier min/max |
| `internal/workflow/nodes/data_loader.go` | source → select: `{Options:["csv","akshare"]}` |
| ... 其他 90+ 节点 | 按需补全约束 |

### 前端

| 文件 | 改动 |
|---|---|
| `frontend/src/workflow/canvas/CustomNode.vue` | 重写 param 渲染：type-based widgets + connect-aware 隐藏 |
| `frontend/src/workflow/PropertyPanel.vue` | 删除 `visibleParams` 区域，简化 |
| `frontend/src/stores/workflow.ts` | `nodeMetaCache` 存储完整 params schema；新增 `hasConnectedPort(paramName)` |

## Acceptance Criteria

- [ ] Go `ParamDef` 扩展字段到位，所有现有节点补全 min/max/step/options
- [ ] `ListNodes()` 返回完整 schema，前端正确解析
- [ ] INT 参数渲染为数字步进控件，min/max/step 生效
- [ ] FLOAT 参数渲染为数字步进控件，支持小数步进
- [ ] STRING 参数渲染为文本输入框
- [ ] BOOL 参数渲染为 toggle 开关
- [ ] select 参数渲染为下拉框，Options 为选项列表
- [ ] 同名 port 连线时 widget 自动隐藏，断线后重新出现
- [ ] 参数值变更实时写入 `node.data.params`
- [ ] PropertyPanel 不再显示参数编辑区域
- [ ] 所有修改通过全量 lint + typecheck + test

## Risks / Trade-offs

- **风险**：90+ 节点补全 min/max 工作量较大，可以分批做（先改常用节点如 sma/macd/rsi/bollinger/data_loader，其余保持 `nil` 即无约束）
- **权衡**：ComfyUI 的 `forceInput` 和 `defaultInput` 逻辑（widget 和端口共存）暂时不实现，简化为一对一互斥
- **兼容性**：现有保存的工作流 JSON 无需迁移 — param 值和之前一样存在 `node.data.params`
- **性能**：每个卡片渲染多个 widget 可能增加 DOM 节点数，但 QuantFlow 工作流通常 < 50 个节点，无性能问题
