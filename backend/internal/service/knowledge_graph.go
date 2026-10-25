package service

import (
	"fmt"
	"strings"
)

// KnowledgePointSeed 知识点种子：Code 是稳定标识，Keywords 用于把已有资源自动挂到该知识点下。
type KnowledgePointSeed struct {
	Code           string
	Name           string
	Description    string
	Keywords       []string
	DisciplineCode string
}

// DisciplineSeed 学科种子：主页面按学科组织，知识点挂在学科下。
// Keywords 同时用于「学科资源匹配」和「无资源时自动爬取的搜索词」。
type DisciplineSeed struct {
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

// DefaultDisciplines 主要学科：覆盖数学、自然科学、计算机、人文社科、医学、工程与艺术。
// 顺序即主页展示顺序；即使某学科暂无资源也保留入口，进入后自动去 B 站抓取。
var DefaultDisciplines = []DisciplineSeed{
	{Code: "math", Name: "数学", Description: "分析、代数、几何、概率与计算数学", Keywords: []string{"数学", "高等数学", "数学分析", "线性代数"}},
	{Code: "physics", Name: "物理学", Description: "力学、电磁学、热学、光学与量子物理", Keywords: []string{"物理", "大学物理", "力学", "电磁学"}},
	{Code: "chemistry", Name: "化学", Description: "无机、有机、物理化学与分析化学", Keywords: []string{"化学", "有机化学", "无机化学"}},
	{Code: "astronomy", Name: "天文学", Description: "天体物理、宇宙学与天文观测", Keywords: []string{"天文", "天体物理"}},
	{Code: "geography", Name: "地理学", Description: "自然地理与人文地理", Keywords: []string{"地理", "自然地理"}},
	{Code: "biology", Name: "生物学", Description: "细胞、遗传、生化与生态", Keywords: []string{"生物", "细胞生物学", "遗传学"}},
	{Code: "computer", Name: "计算机科学", Description: "数据结构、操作系统、网络与数据库", Keywords: []string{"计算机", "数据结构", "操作系统"}},
	{Code: "ai", Name: "人工智能", Description: "机器学习、深度学习与大模型", Keywords: []string{"人工智能", "机器学习", "深度学习"}},
	{Code: "economics", Name: "经济学", Description: "微观、宏观、计量与金融", Keywords: []string{"经济学", "微观经济学", "宏观经济学"}},
	{Code: "management", Name: "管理学", Description: "管理学原理、市场营销与组织行为", Keywords: []string{"管理学", "市场营销"}},
	{Code: "law", Name: "法学", Description: "法理、民法、刑法与程序法", Keywords: []string{"法学", "民法", "刑法"}},
	{Code: "education", Name: "教育学", Description: "教育原理、课程教学与教育心理", Keywords: []string{"教育学", "教育心理学"}},
	{Code: "psychology", Name: "心理学", Description: "普通、发展与认知心理学", Keywords: []string{"心理学", "认知心理学"}},
	{Code: "literature", Name: "文学", Description: "中国古代、现当代与外国文学", Keywords: []string{"文学", "中国古代文学"}},
	{Code: "history", Name: "历史学", Description: "中国古代史、近现代史与世界史", Keywords: []string{"历史", "中国史", "世界史"}},
	{Code: "philosophy", Name: "哲学", Description: "中国哲学、西方哲学、逻辑与伦理", Keywords: []string{"哲学", "西方哲学"}},
	{Code: "foreign", Name: "外国语言", Description: "大学英语、雅思、翻译与语言学", Keywords: []string{"英语", "外语", "雅思"}},
	{Code: "medicine", Name: "医学", Description: "解剖、生理、病理与临床基础", Keywords: []string{"医学", "解剖学", "生理学"}},
	{Code: "engineering", Name: "工学", Description: "工程力学、电路与材料科学", Keywords: []string{"工程", "电路", "材料科学"}},
	{Code: "art", Name: "艺术学", Description: "音乐理论、美术史与影视艺术", Keywords: []string{"艺术", "音乐", "美术"}},
	{Code: "environment", Name: "环境科学", Description: "环境科学导论、环境化学与生态工程", Keywords: []string{"环境科学", "环境工程"}},
}

// DefaultKnowledgePoints 默认知识点集合（挂在学科下）。
//
// 数学与 AI 保留原有先修链条；其余学科先给出通识/主干课程，
// 资源不足时进入学科页会自动去 B 站抓取并挂到这些知识点下。
var DefaultKnowledgePoints = []KnowledgePointSeed{
	// 数学
	{Code: "calculus", Name: "高等数学", DisciplineCode: "math", Description: "一元与多元微积分基础，理工科第一门数学课", Keywords: []string{"高等数学", "微积分", "高数"}},
	{Code: "linear-algebra", Name: "线性代数", DisciplineCode: "math", Description: "向量空间、矩阵、特征值与线性变换", Keywords: []string{"线性代数", "线代"}},
	{Code: "analytic-geometry", Name: "解析几何", DisciplineCode: "math", Description: "用代数方法研究空间图形，几何直观与坐标运算", Keywords: []string{"解析几何"}},
	{Code: "probability", Name: "概率论与数理统计", DisciplineCode: "math", Description: "随机变量、分布、估计与假设检验", Keywords: []string{"概率论", "数理统计", "概率统计"}},
	{Code: "mathematical-analysis", Name: "数学分析", DisciplineCode: "math", Description: "极限、连续、微分与积分的严格理论", Keywords: []string{"数学分析"}},
	{Code: "advanced-algebra", Name: "高等代数", DisciplineCode: "math", Description: "多项式、矩阵与线性空间的理论体系", Keywords: []string{"高等代数", "高代"}},
	{Code: "abstract-algebra", Name: "抽象代数", DisciplineCode: "math", Description: "群、环、域等代数结构", Keywords: []string{"抽象代数", "近世代数", "群论", "群表示论"}},
	{Code: "complex-analysis", Name: "复变函数与积分变换", DisciplineCode: "math", Description: "解析函数、留数、傅里叶与拉普拉斯变换", Keywords: []string{"复变函数", "积分变换"}},
	{Code: "real-analysis", Name: "实变函数", DisciplineCode: "math", Description: "测度、可测函数与勒贝格积分", Keywords: []string{"实变函数", "实分析"}},
	{Code: "topology", Name: "拓扑学", DisciplineCode: "math", Description: "开集、连续映射与空间的不变量", Keywords: []string{"拓扑"}},
	{Code: "functional-analysis", Name: "泛函分析", DisciplineCode: "math", Description: "无穷维空间上的算子与变分方法", Keywords: []string{"泛函分析", "变分原理"}},
	{Code: "numerical-analysis", Name: "数值分析", DisciplineCode: "math", Description: "数值逼近、数值积分与线性方程组的数值解法", Keywords: []string{"数值分析", "数值计算"}},

	// 计算机 / 人工智能
	{Code: "data-structures", Name: "数据结构与算法", DisciplineCode: "computer", Description: "线性表、树、图与常见算法设计", Keywords: []string{"数据结构", "算法与数据结构"}},
	{Code: "operating-systems", Name: "操作系统", DisciplineCode: "computer", Description: "进程、内存、文件系统与并发", Keywords: []string{"操作系统"}},
	{Code: "computer-networks", Name: "计算机网络", DisciplineCode: "computer", Description: "TCP/IP、路由、传输层与应用层", Keywords: []string{"计算机网络"}},
	{Code: "database-systems", Name: "数据库系统", DisciplineCode: "computer", Description: "关系模型、SQL、事务与索引", Keywords: []string{"数据库", "SQL"}},
	{Code: "machine-learning", Name: "机器学习", DisciplineCode: "ai", Description: "监督学习、无监督学习与模型评估", Keywords: []string{"机器学习", "机器学习入门"}},
	{Code: "deep-learning", Name: "深度学习", DisciplineCode: "ai", Description: "神经网络、反向传播与深度模型", Keywords: []string{"深度学习", "神经网络", "大模型"}},

	// 物理
	{Code: "college-physics", Name: "大学物理", DisciplineCode: "physics", Description: "力、热、光、电的基础物理图景", Keywords: []string{"大学物理", "普通物理"}},
	{Code: "mechanics", Name: "力学", DisciplineCode: "physics", Description: "牛顿力学、分析力学与刚体运动", Keywords: []string{"力学", "理论力学"}},
	{Code: "electromagnetism", Name: "电磁学", DisciplineCode: "physics", Description: "静电场、磁场、电磁感应与麦克斯韦方程", Keywords: []string{"电磁学", "电动力学"}},
	{Code: "thermodynamics", Name: "热力学与统计物理", DisciplineCode: "physics", Description: "热力学定律、熵与统计分布", Keywords: []string{"热力学", "统计物理"}},
	{Code: "quantum-mechanics", Name: "量子力学", DisciplineCode: "physics", Description: "波函数、算符、氢原子与量子测量", Keywords: []string{"量子力学"}},
	{Code: "optics", Name: "光学", DisciplineCode: "physics", Description: "几何光学、波动光学与激光", Keywords: []string{"光学"}},

	// 化学
	{Code: "inorganic-chemistry", Name: "无机化学", DisciplineCode: "chemistry", Description: "元素化学、配位化学与结构", Keywords: []string{"无机化学"}},
	{Code: "organic-chemistry", Name: "有机化学", DisciplineCode: "chemistry", Description: "官能团、反应机理与合成", Keywords: []string{"有机化学"}},
	{Code: "physical-chemistry", Name: "物理化学", DisciplineCode: "chemistry", Description: "化学热力学、动力学与电化学", Keywords: []string{"物理化学"}},
	{Code: "analytical-chemistry", Name: "分析化学", DisciplineCode: "chemistry", Description: "滴定、光谱与色谱分析", Keywords: []string{"分析化学"}},

	// 生物
	{Code: "cell-biology", Name: "细胞生物学", DisciplineCode: "biology", Description: "细胞结构、代谢与信号转导", Keywords: []string{"细胞生物学"}},
	{Code: "genetics", Name: "遗传学", DisciplineCode: "biology", Description: "孟德尔遗传、基因表达与变异", Keywords: []string{"遗传学"}},
	{Code: "biochemistry", Name: "生物化学", DisciplineCode: "biology", Description: "蛋白质、酶与代谢通路", Keywords: []string{"生物化学"}},
	{Code: "ecology", Name: "生态学", DisciplineCode: "biology", Description: "种群、群落与生态系统", Keywords: []string{"生态学"}},

	// 天文 / 地理 / 环境
	{Code: "intro-astronomy", Name: "天文学导论", DisciplineCode: "astronomy", Description: "星空、恒星演化与宇宙学基础", Keywords: []string{"天文学", "天文"}},
	{Code: "astrophysics", Name: "天体物理", DisciplineCode: "astronomy", Description: "恒星结构、星系与宇宙学", Keywords: []string{"天体物理"}},
	{Code: "physical-geography", Name: "自然地理学", DisciplineCode: "geography", Description: "地貌、气候、水文与土壤", Keywords: []string{"自然地理"}},
	{Code: "human-geography", Name: "人文地理学", DisciplineCode: "geography", Description: "人口、城市、经济与文化地理", Keywords: []string{"人文地理"}},
	{Code: "intro-environmental-science", Name: "环境科学导论", DisciplineCode: "environment", Description: "环境系统、污染与可持续发展", Keywords: []string{"环境科学"}},
	{Code: "environmental-chemistry", Name: "环境化学", DisciplineCode: "environment", Description: "大气、水体与土壤中的化学过程", Keywords: []string{"环境化学"}},

	// 经济 / 管理
	{Code: "microeconomics", Name: "微观经济学", DisciplineCode: "economics", Description: "供求、消费者与厂商理论", Keywords: []string{"微观经济学"}},
	{Code: "macroeconomics", Name: "宏观经济学", DisciplineCode: "economics", Description: "GDP、通胀、失业与宏观政策", Keywords: []string{"宏观经济学"}},
	{Code: "finance", Name: "金融学", DisciplineCode: "economics", Description: "货币、投资与金融市场", Keywords: []string{"金融学", "金融"}},
	{Code: "management", Name: "管理学", DisciplineCode: "management", Description: "计划、组织、领导与控制", Keywords: []string{"管理学"}},
	{Code: "marketing", Name: "市场营销", DisciplineCode: "management", Description: "市场分析、品牌与消费者行为", Keywords: []string{"市场营销"}},

	// 法学 / 教育 / 心理
	{Code: "jurisprudence", Name: "法理学", DisciplineCode: "law", Description: "法的概念、渊源与法律解释", Keywords: []string{"法理学"}},
	{Code: "civil-law", Name: "民法", DisciplineCode: "law", Description: "民事主体、物权、合同与侵权", Keywords: []string{"民法"}},
	{Code: "criminal-law", Name: "刑法", DisciplineCode: "law", Description: "犯罪构成、刑罚与分则", Keywords: []string{"刑法"}},
	{Code: "pedagogy", Name: "教育学原理", DisciplineCode: "education", Description: "教育本质、目的与制度", Keywords: []string{"教育学"}},
	{Code: "educational-psychology", Name: "教育心理学", DisciplineCode: "education", Description: "学习理论、动机与教学心理", Keywords: []string{"教育心理学"}},
	{Code: "general-psychology", Name: "普通心理学", DisciplineCode: "psychology", Description: "感知、记忆、情绪与人格", Keywords: []string{"普通心理学", "心理学"}},
	{Code: "developmental-psychology", Name: "发展心理学", DisciplineCode: "psychology", Description: "认知与社会性发展阶段", Keywords: []string{"发展心理学"}},
	{Code: "cognitive-psychology", Name: "认知心理学", DisciplineCode: "psychology", Description: "注意、记忆、语言与决策", Keywords: []string{"认知心理学"}},

	// 文学 / 历史 / 哲学
	{Code: "ancient-chinese-literature", Name: "中国古代文学", DisciplineCode: "literature", Description: "先秦至明清文学史与作品", Keywords: []string{"中国古代文学"}},
	{Code: "modern-chinese-literature", Name: "中国现当代文学", DisciplineCode: "literature", Description: "现当代文学思潮与作家作品", Keywords: []string{"现当代文学"}},
	{Code: "world-literature", Name: "外国文学", DisciplineCode: "literature", Description: "欧美与东方文学经典", Keywords: []string{"外国文学"}},
	{Code: "ancient-chinese-history", Name: "中国古代史", DisciplineCode: "history", Description: "先秦至明清的政治与社会", Keywords: []string{"中国古代史"}},
	{Code: "modern-chinese-history", Name: "中国近现代史", DisciplineCode: "history", Description: "近代转型与现代中国", Keywords: []string{"中国近现代史", "近代史"}},
	{Code: "world-history", Name: "世界史", DisciplineCode: "history", Description: "全球文明与近现代世界", Keywords: []string{"世界史"}},
	{Code: "chinese-philosophy", Name: "中国哲学", DisciplineCode: "philosophy", Description: "先秦诸子、理学与心学", Keywords: []string{"中国哲学"}},
	{Code: "western-philosophy", Name: "西方哲学", DisciplineCode: "philosophy", Description: "古希腊至德国古典哲学", Keywords: []string{"西方哲学"}},
	{Code: "logic", Name: "逻辑学", DisciplineCode: "philosophy", Description: "命题逻辑、谓词逻辑与论证", Keywords: []string{"逻辑学"}},
	{Code: "ethics", Name: "伦理学", DisciplineCode: "philosophy", Description: "规范伦理、元伦理与应用伦理", Keywords: []string{"伦理学"}},

	// 外语
	{Code: "college-english", Name: "大学英语", DisciplineCode: "foreign", Description: "词汇、语法、阅读与写作", Keywords: []string{"大学英语"}},
	{Code: "ielts", Name: "雅思备考", DisciplineCode: "foreign", Description: "听力、口语、阅读、写作四科备考", Keywords: []string{"雅思"}},
	{Code: "translation", Name: "翻译", DisciplineCode: "foreign", Description: "英汉互译技巧与实务", Keywords: []string{"翻译"}},

	// 医学 / 工学 / 艺术
	{Code: "anatomy", Name: "解剖学", DisciplineCode: "medicine", Description: "人体系统与局部解剖", Keywords: []string{"解剖学"}},
	{Code: "physiology", Name: "生理学", DisciplineCode: "medicine", Description: "人体各系统的功能与调节", Keywords: []string{"生理学"}},
	{Code: "pathology", Name: "病理学", DisciplineCode: "medicine", Description: "疾病发生机制与病理变化", Keywords: []string{"病理学"}},
	{Code: "engineering-mechanics", Name: "工程力学", DisciplineCode: "engineering", Description: "静力学、材料力学与结构", Keywords: []string{"工程力学"}},
	{Code: "circuit-theory", Name: "电路原理", DisciplineCode: "engineering", Description: "电路分析、网络定理与暂态", Keywords: []string{"电路", "电路原理"}},
	{Code: "materials-science", Name: "材料科学基础", DisciplineCode: "engineering", Description: "晶体结构、相图与材料性能", Keywords: []string{"材料科学"}},
	{Code: "music-theory", Name: "音乐理论", DisciplineCode: "art", Description: "乐理、和声与曲式", Keywords: []string{"音乐理论", "乐理"}},
	{Code: "art-history", Name: "美术史", DisciplineCode: "art", Description: "中外美术史与作品鉴赏", Keywords: []string{"美术史"}},
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
