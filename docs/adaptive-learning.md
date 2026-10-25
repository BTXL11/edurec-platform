# 知识图谱与自适应学习

面向「前端知识点导航 + 后端自适应推荐」的完整链路：

```
学科（16 个知识点） ──先修边──► 概念层（EMBEDD-ER / MOOCCube）
        │                              │
        └──── 资源关联（370+）◄────────┘
                       │
        初始问卷 / 日常行为 / 完成路径
                       ▼
                 学习画像（掌握度）
                       ▼
        目标知识点 → 先修闭包 → 学习路径（done/current/locked）
                       ▼
              只推荐当前一节未掌握的资源
```

## 1. 两层知识图谱

### 学科层

- 节点：`knowledge_points`，16 个学科（高等数学、线性代数、数学分析、机器学习……）。
- 边：`knowledge_point_edges`，19 条先修关系（如 高等数学 → 数学分析 → 实变函数）。
- 资源关联：`resource_knowledge_points`，按标题/标签/描述与学科关键词的加权匹配
  （标题 1.0、标签 0.6、描述 0.3，阈值 0.5），当前 370+ 条。
- 层级由先修边拓扑推导，**有环直接报错**（学科层必须严格无环）。

### 概念层（EMBEDD-ER / MOOCCube）

- 数据来源：`E:\work\learn\EMBEDD-ER\data\MOOCCube`（外部仓库，不入库）。
- 规模：425 个概念、1027 条先修边（数学 + 计算机科学技术）。
- 概念没有学科标签，导入时通过「课程 → 概念」反推：
  课程名按学科关键词归类 → 概念按所属课程投票 → 票数最多的学科即归属。
  当前 206 个概念能归入学科。
- MOOCCube 原始先修图**含环**，概念层级用 SCC 缩点的容错算法（环内同级、环外递增）。

### 数据源路径与重建

路径走配置 `knowledge.source_dir`（默认相对 `backend/`，也支持环境变量
`KNOWLEDGE_SOURCE_DIR` 覆盖）。当前本地指向外部仓库：

```yaml
knowledge:
  source_dir: "E:/work/learn/EMBEDD-ER/data/MOOCCube"
```

迁入 engine 仓库后只改这一处路径，**不改代码**：

```powershell
.\scripts\set-kg-source.ps1 -Show
.\scripts\set-kg-source.ps1 -Path 'D:\engine\data\MOOCCube'
```

图谱规则调整后整体重建（只清图谱派生表，资源/用户/行为不动）：

```powershell
.\scripts\rebuild-kg.ps1              # 全量重建
.\scripts\rebuild-kg.ps1 -DryRun      # 只统计
.\scripts\rebuild-kg.ps1 -KeepOld     # 增量
```

## 2. 学习画像

掌握度（0-1）取以下三者的较大值：

| 来源 | 规则 |
|---|---|
| 初始问卷 | 自评 0/1/2/3 → 0 / 0.33 / 0.67 / 1 |
| 日常行为 | 行为次数 × 0.1，**上限 0.6**（仅看视频不应算已掌握） |
| 完成路径 | 完成某一节后该知识点掌握度置 1 |

掌握度 **≥ 0.8** 视为已掌握：生成路径时直接标 `done`，不再作为待学内容。

## 3. 学习路径算法

1. **先修闭包**：从目标知识点沿先修边反向收集所有前置节点；
2. **拓扑排序**：闭包内按入度出队，同层按 `level`、`id` 稳定排序；有环报错；
3. **生成步骤**：已掌握 → `done`，第一个未掌握 → `current`，其余 → `locked`；
4. **逐节解锁**：只能完成 `current` 步骤，完成后下一节变 `current`；
   完成 `locked` 步骤会被拒绝；全部完成时路径状态置 `completed`。

## 4. API

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/knowledge-points` | 学科列表（资源数、概念数） |
| GET | `/api/v1/knowledge-points/:code` | 学科图 + 概念子图 + 资源 |
| GET | `/api/v1/knowledge-points/:code/resources` | 学科资源分页 |
| GET | `/api/v1/learning-profile` | 当前用户知识画像 |
| POST | `/api/v1/learning-profile/questionnaire` | 提交问卷 `{answers:{code:0-3}}` |
| POST | `/api/v1/learning-paths` | 生成路径 `{target:"machine-learning"}` |
| GET | `/api/v1/learning-paths/:id` | 路径详情 |
| POST | `/api/v1/learning-paths/:id/complete` | 完成当前节 `{index:1}` |
| GET | `/api/v1/learning-paths/:id/resources` | 当前节资源 |

生成路径示例（问卷：高等数学=3、线性代数=2、概率论=1）：

```json
{
  "target_name": "机器学习",
  "steps": [
    {"name":"高等数学","status":"done","mastery":1},
    {"name":"线性代数","status":"current","mastery":0.67},
    {"name":"数据结构与算法","status":"locked","mastery":0},
    {"name":"概率论与数理统计","status":"locked","mastery":0.33},
    {"name":"机器学习","status":"locked","mastery":0}
  ],
  "current_index": 1,
  "status": "active"
}
```

## 5. 前端页面

| 页面 | 路径 | 说明 |
|---|---|---|
| 知识点导航 | `/knowledge` | 按层级分组、关键词过滤、问卷入口 |
| 知识点详情 | `/knowledge/:code` | SVG 先修图、概念标签、资源、生成学习路径 |
| 初始问卷 | `/questionnaire` | 0-3 自评，回填已有掌握度 |
| 学习路径 | `/learning-path/:id` | 进度、步骤状态、当前节资源、完成解锁 |

图谱视图用纯 SVG + `layoutKnowledgeGraph` 纯函数实现（按 level 分列、同层居中），
不引入图形库。

## 6. 关键帧与封面

资源详情页还展示 B 站关键帧（雪碧图 CSS 裁切），详见
[bilibili-import.md](bilibili-import.md#关键帧与封面资源特征)。

## 7. 已知限制

- 概念层只覆盖 MOOCCube 有课程名的学科；数学分析、高等代数、泛函分析等暂无概念，
  页面会明确提示。
- 行为侧掌握度是简单线性估计（0.1/次，上限 0.6），后续可换成引擎侧的掌握度模型。
- 学习路径尚未合并进首页推荐流；首页仍走 engine 推荐，路径当前节资源在路径页展示。
- 概念归属依赖课程名关键词，课程名缺失时该概念不归属任何学科。
