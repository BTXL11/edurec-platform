package service

import "strings"

// MatchSubjectCode 把课程名/标题按学科关键词归到学科，返回学科 code；无法归类返回空串。
// 多个关键词同时命中时取最长的一个（「线性代数」优先于「代数」），
// 长度相同按 DefaultKnowledgePoints 的顺序，保证结果稳定。
func MatchSubjectCode(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return ""
	}

	bestCode := ""
	bestLen := 0
	for _, point := range DefaultKnowledgePoints {
		for _, keyword := range point.Keywords {
			kw := strings.ToLower(strings.TrimSpace(keyword))
			if kw == "" || !strings.Contains(name, kw) {
				continue
			}
			if len(kw) > bestLen {
				bestLen = len(kw)
				bestCode = point.Code
			}
		}
	}
	return bestCode
}

// AssignConceptSubjects 给概念分配学科：概念经由「课程 -> 概念」的包含关系投票，
// 课程先按名称归到学科，一个概念出现最多的学科即为其归属。
//
// 这是「学科入口 + 概念图作下层」的关键一步：MOOCCube 的概念本身没有学科标签，
// 但它所属的课程有名称，课程名能稳定归到学科。票数相同时按学科定义顺序取第一个，
// 保证重复导入结果一致。
func AssignConceptSubjects(
	conceptIDs []string,
	courseConcepts map[string][]string,
	courseNames map[string]string,
) map[string]string {
	subjectOrder := make(map[string]int, len(DefaultKnowledgePoints))
	for i, point := range DefaultKnowledgePoints {
		subjectOrder[point.Code] = i
	}

	votes := make(map[string]map[string]int, len(conceptIDs))
	for courseID, concepts := range courseConcepts {
		subject := MatchSubjectCode(courseNames[courseID])
		if subject == "" {
			continue
		}
		for _, concept := range concepts {
			if votes[concept] == nil {
				votes[concept] = make(map[string]int, 2)
			}
			votes[concept][subject]++
		}
	}

	assigned := make(map[string]string, len(conceptIDs))
	for _, concept := range conceptIDs {
		counts := votes[concept]
		if len(counts) == 0 {
			continue
		}
		bestSubject := ""
		bestVotes := -1
		for subject, count := range counts {
			if count > bestVotes || (count == bestVotes && subjectOrder[subject] < subjectOrder[bestSubject]) {
				bestVotes = count
				bestSubject = subject
			}
		}
		if bestSubject != "" {
			assigned[concept] = bestSubject
		}
	}
	return assigned
}
