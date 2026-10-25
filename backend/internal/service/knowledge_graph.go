package service

import (
	"fmt"
	"strings"
)

// KnowledgePointSeed 知识点种子：Code 是稳定标识，Keywords 用于把已有资源自动挂到该知识点下。
type KnowledgePointSeed struct {
	Code        string
	Name        string
	Description string
	Keywords    []string
}

// KnowledgeEdgeSeed 先修关系种子：From 是 To 的先修。
type KnowledgeEdgeSeed struct {
	From string
	To   string
}

// DefaultKnowledgePoints 默认知识点集合。
//
// 覆盖面按仓库里实际存在的 B 站资源设计（数学分析、高等代数、泛函分析、实变、拓扑、机器学习、雅思等），
// 只收录能形成「先修 -> 进阶」链条的学科，避免把零散主题硬凑成图。
var DefaultKnowledgePoints = []KnowledgePointSeed{
	{Code: "calculus", Name: "高等数学", Description: "一元与多元微积分基础，理工科第一门数学课", Keywords: []string{"高等数学", "微积分", "高数"}},
	{Code: "linear-algebra", Name: "线性代数", Description: "向量空间、矩阵、特征值与线性变换", Keywords: []string{"线性代数", "线代"}},
	{Code: "analytic-geometry", Name: "解析几何", Description: "用代数方法研究空间图形，几何直观与坐标运算", Keywords: []string{"解析几何"}},
	{Code: "probability", Name: "概率论与数理统计", Description: "随机变量、分布、估计与假设检验", Keywords: []string{"概率论", "数理统计", "概率统计"}},
	{Code: "mathematical-analysis", Name: "数学分析", Description: "极限、连续、微分与积分的严格理论", Keywords: []string{"数学分析"}},
	{Code: "advanced-algebra", Name: "高等代数", Description: "多项式、矩阵与线性空间的理论体系", Keywords: []string{"高等代数", "高代"}},
	{Code: "abstract-algebra", Name: "抽象代数", Description: "群、环、域等代数结构", Keywords: []string{"抽象代数", "近世代数", "群论", "群表示论"}},
	{Code: "complex-analysis", Name: "复变函数与积分变换", Description: "解析函数、留数、傅里叶与拉普拉斯变换", Keywords: []string{"复变函数", "积分变换"}},
	{Code: "real-analysis", Name: "实变函数", Description: "测度、可测函数与勒贝格积分", Keywords: []string{"实变函数", "实分析"}},
	{Code: "topology", Name: "拓扑学", Description: "开集、连续映射与空间的不变量", Keywords: []string{"拓扑"}},
	{Code: "functional-analysis", Name: "泛函分析", Description: "无穷维空间上的算子与变分方法", Keywords: []string{"泛函分析", "变分原理"}},
	{Code: "numerical-analysis", Name: "数值分析", Description: "数值逼近、数值积分与线性方程组的数值解法", Keywords: []string{"数值分析", "数值计算"}},
	{Code: "data-structures", Name: "数据结构与算法", Description: "线性表、树、图与常见算法设计", Keywords: []string{"数据结构", "算法与数据结构"}},
	{Code: "machine-learning", Name: "机器学习", Description: "监督学习、无监督学习与模型评估", Keywords: []string{"机器学习", "机器学习入门"}},
	{Code: "deep-learning", Name: "深度学习", Description: "神经网络、反向传播与深度模型", Keywords: []string{"深度学习", "神经网络", "大模型"}},
	{Code: "ielts", Name: "雅思备考", Description: "听力、口语、阅读、写作四科备考", Keywords: []string{"雅思"}},
}

// DefaultKnowledgeEdges 默认先修关系。数学课按「基础 -> 分析 -> 专题 -> 应用」排列，
// 机器学习依赖线性代数、概率论与高等数学，深度学习再依赖机器学习。
var DefaultKnowledgeEdges = []KnowledgeEdgeSeed{
	{From: "calculus", To: "mathematical-analysis"},
	{From: "calculus", To: "probability"},
	{From: "analytic-geometry", To: "advanced-algebra"},
	{From: "linear-algebra", To: "advanced-algebra"},
	{From: "advanced-algebra", To: "abstract-algebra"},
	{From: "mathematical-analysis", To: "complex-analysis"},
	{From: "mathematical-analysis", To: "real-analysis"},
	{From: "mathematical-analysis", To: "functional-analysis"},
	{From: "linear-algebra", To: "functional-analysis"},
	{From: "real-analysis", To: "functional-analysis"},
	{From: "real-analysis", To: "topology"},
	{From: "mathematical-analysis", To: "numerical-analysis"},
	{From: "linear-algebra", To: "numerical-analysis"},
	{From: "linear-algebra", To: "machine-learning"},
	{From: "probability", To: "machine-learning"},
	{From: "calculus", To: "machine-learning"},
	{From: "machine-learning", To: "deep-learning"},
	{From: "linear-algebra", To: "deep-learning"},
	{From: "data-structures", To: "machine-learning"},
}

// ComputeLevels 按先修关系推导每个知识点的层级：无先修为 1，其余为「先修层级最大值 + 1」。
// 这样同一层的知识点可以并排展示，层级表示学习顺序的远近。
// 出现环时返回错误——有环的图无法生成合法学习路径，必须在种子阶段就拦住。
func ComputeLevels(codes []string, edges []KnowledgeEdgeSeed) (map[string]int, error) {
	level := make(map[string]int, len(codes))
	inDegree := make(map[string]int, len(codes))
	next := make(map[string][]string, len(codes))

	for _, code := range codes {
		level[code] = 1
	}

	seen := make(map[string]struct{}, len(edges))
	for _, edge := range edges {
		if edge.From == edge.To {
			return nil, fmt.Errorf("知识点 %q 的先修关系指向自身", edge.From)
		}
		key := edge.From + "->" + edge.To
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		next[edge.From] = append(next[edge.From], edge.To)
		inDegree[edge.To]++
	}

	queue := make([]string, 0, len(codes))
	for _, code := range codes {
		if inDegree[code] == 0 {
			queue = append(queue, code)
		}
	}

	processed := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		processed++

		for _, to := range next[current] {
			if level[current]+1 > level[to] {
				level[to] = level[current] + 1
			}
			inDegree[to]--
			if inDegree[to] == 0 {
				queue = append(queue, to)
			}
		}
	}

	if processed != len(codes) {
		return nil, fmt.Errorf("知识点先修关系存在环，无法推导层级")
	}
	return level, nil
}

// ComputeLevelsTolerant 允许图中存在环：先求强连通分量（SCC）缩点，再在缩点后的
// 有向无环图上分层。同一环内的概念共用一个层级。
//
// MOOCCube 的概念先修数据含有环（例如 A 依赖 B、B 又依赖 A），学科层的严格分层
// 无法处理；概念层必须容错，否则整张图都导不进来。
func ComputeLevelsTolerant(codes []string, edges []KnowledgeEdgeSeed) map[string]int {
	index := make(map[string]int, len(codes))
	low := make(map[string]int, len(codes))
	onStack := make(map[string]bool, len(codes))
	comp := make(map[string]int, len(codes))
	adj := make(map[string][]string, len(codes))
	order := make([]string, 0, len(codes))

	for _, code := range codes {
		order = append(order, code)
	}
	seen := make(map[string]struct{}, len(edges))
	for _, edge := range edges {
		if edge.From == edge.To {
			continue
		}
		key := edge.From + "\x00" + edge.To
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		adj[edge.From] = append(adj[edge.From], edge.To)
	}

	counter := 0
	compID := 0
	var stack []string

	var strongConnect func(v string)
	strongConnect = func(v string) {
		index[v] = counter
		low[v] = counter
		counter++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range adj[v] {
			if _, visited := index[w]; !visited {
				strongConnect(w)
				if low[w] < low[v] {
					low[v] = low[w]
				}
			} else if onStack[w] && index[w] < low[v] {
				low[v] = index[w]
			}
		}

		if low[v] == index[v] {
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp[w] = compID
				if w == v {
					break
				}
			}
			compID++
		}
	}

	for _, code := range order {
		if _, visited := index[code]; !visited {
			strongConnect(code)
		}
	}

	// 缩点后的图
	compEdges := make(map[int]map[int]struct{}, compID)
	inDegree := make([]int, compID)
	for _, edge := range edges {
		fromComp, okFrom := comp[edge.From]
		toComp, okTo := comp[edge.To]
		if !okFrom || !okTo || fromComp == toComp {
			continue
		}
		if compEdges[fromComp] == nil {
			compEdges[fromComp] = make(map[int]struct{}, 2)
		}
		if _, exists := compEdges[fromComp][toComp]; exists {
			continue
		}
		compEdges[fromComp][toComp] = struct{}{}
		inDegree[toComp]++
	}

	compLevel := make([]int, compID)
	for i := range compLevel {
		compLevel[i] = 1
	}
	queue := make([]int, 0, compID)
	for id := 0; id < compID; id++ {
		if inDegree[id] == 0 {
			queue = append(queue, id)
		}
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for next := range compEdges[current] {
			if compLevel[current]+1 > compLevel[next] {
				compLevel[next] = compLevel[current] + 1
			}
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	levels := make(map[string]int, len(codes))
	for _, code := range codes {
		levels[code] = compLevel[comp[code]]
	}
	return levels
}
//
// 标题命中记 1.0、标签命中 0.6、描述命中 0.3，同一个关键词只取最高命中的字段；
// 命中越多越接近 1。调用方用阈值（默认 0.5）过滤，即「标题或标签命中」才建立关联，
// 避免描述里偶然出现的词把资源挂到无关知识点上。
func MatchScore(title string, tags []string, description string, keywords []string) float64 {
	if len(keywords) == 0 {
		return 0
	}
	title = strings.ToLower(title)
	description = strings.ToLower(description)
	tagText := strings.ToLower(strings.Join(tags, " "))

	score := 0.0
	for _, keyword := range keywords {
		kw := strings.ToLower(strings.TrimSpace(keyword))
		if kw == "" {
			continue
		}
		switch {
		case strings.Contains(title, kw):
			score += 1.0
		case strings.Contains(tagText, kw):
			score += 0.6
		case strings.Contains(description, kw):
			score += 0.3
		}
	}
	if score > 1 {
		score = 1
	}
	return score
}
