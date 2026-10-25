package service_test

import (
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

func TestComputeLevelsChain(t *testing.T) {
	codes := []string{"a", "b", "c"}
	edges := []service.KnowledgeEdgeSeed{
		{From: "a", To: "b"},
		{From: "b", To: "c"},
	}

	levels, err := service.ComputeLevels(codes, edges)
	if err != nil {
		t.Fatalf("ComputeLevels() error = %v", err)
	}

	want := map[string]int{"a": 1, "b": 2, "c": 3}
	for code, level := range want {
		if levels[code] != level {
			t.Errorf("level[%s] = %d, want %d", code, levels[code], level)
		}
	}
}

// 菱形依赖：同层节点应取「先修最大层级 + 1」，而不是简单的层数累加
func TestComputeLevelsDiamond(t *testing.T) {
	codes := []string{"root", "left", "right", "join"}
	edges := []service.KnowledgeEdgeSeed{
		{From: "root", To: "left"},
		{From: "root", To: "right"},
		{From: "left", To: "join"},
		{From: "right", To: "join"},
	}

	levels, err := service.ComputeLevels(codes, edges)
	if err != nil {
		t.Fatalf("ComputeLevels() error = %v", err)
	}

	if levels["join"] != 3 {
		t.Errorf("join level = %d, want 3", levels["join"])
	}
}

func TestComputeLevelsRejectsCycle(t *testing.T) {
	codes := []string{"a", "b"}
	edges := []service.KnowledgeEdgeSeed{
		{From: "a", To: "b"},
		{From: "b", To: "a"},
	}

	if _, err := service.ComputeLevels(codes, edges); err == nil {
		t.Fatal("有环的图应当报错")
	}
}

func TestComputeLevelsRejectsSelfEdge(t *testing.T) {
	codes := []string{"a"}
	edges := []service.KnowledgeEdgeSeed{{From: "a", To: "a"}}

	if _, err := service.ComputeLevels(codes, edges); err == nil {
		t.Fatal("自环应当报错")
	}
}

// 概念图含环时不能整体失败：环内节点同级，环外节点照常分层
func TestComputeLevelsTolerantHandlesCycle(t *testing.T) {
	codes := []string{"a", "b", "c"}
	edges := []service.KnowledgeEdgeSeed{
		{From: "a", To: "b"},
		{From: "b", To: "a"},
		{From: "b", To: "c"},
	}

	levels := service.ComputeLevelsTolerant(codes, edges)

	if levels["a"] != levels["b"] {
		t.Errorf("同环内的 a/b 应同级，got a=%d b=%d", levels["a"], levels["b"])
	}
	if levels["c"] <= levels["b"] {
		t.Errorf("环外后继 c 应晚于环内节点，got b=%d c=%d", levels["b"], levels["c"])
	}

	// 严格版本对同一张图仍应报错（学科层保持严格）
	if _, err := service.ComputeLevels(codes, edges); err == nil {
		t.Fatal("严格分层应当发现有环")
	}
}

func TestComputeLevelsTolerantIgnoresSelfEdge(t *testing.T) {
	levels := service.ComputeLevelsTolerant(
		[]string{"a"},
		[]service.KnowledgeEdgeSeed{{From: "a", To: "a"}},
	)
	if levels["a"] != 1 {
		t.Errorf("a level = %d, want 1", levels["a"])
	}
}

func TestComputeLevelsIgnoresDuplicateEdges(t *testing.T) {
	codes := []string{"a", "b"}
	edges := []service.KnowledgeEdgeSeed{
		{From: "a", To: "b"},
		{From: "a", To: "b"},
	}

	levels, err := service.ComputeLevels(codes, edges)
	if err != nil {
		t.Fatalf("ComputeLevels() error = %v", err)
	}
	if levels["b"] != 2 {
		t.Errorf("b level = %d, want 2", levels["b"])
	}
}

// 默认图必须合法：有环会导致学习路径无法生成
func TestDefaultKnowledgeGraphIsAcyclic(t *testing.T) {
	codes := make([]string, 0, len(service.DefaultKnowledgePoints))
	for _, point := range service.DefaultKnowledgePoints {
		codes = append(codes, point.Code)
	}

	levels, err := service.ComputeLevels(codes, service.DefaultKnowledgeEdges)
	if err != nil {
		t.Fatalf("默认知识点图不合法: %v", err)
	}

	// 抽查几条关键层级：先修链越深，层级越大
	if levels["calculus"] != 1 {
		t.Errorf("calculus level = %d, want 1", levels["calculus"])
	}
	if levels["mathematical-analysis"] <= levels["calculus"] {
		t.Errorf("数学分析应晚于高等数学")
	}
	if levels["functional-analysis"] <= levels["mathematical-analysis"] {
		t.Errorf("泛函分析应晚于数学分析")
	}
	if levels["deep-learning"] <= levels["machine-learning"] {
		t.Errorf("深度学习应晚于机器学习")
	}
}

func TestDefaultKnowledgePointsHaveUniqueCodes(t *testing.T) {
	seen := map[string]struct{}{}
	for _, point := range service.DefaultKnowledgePoints {
		if point.Code == "" || point.Name == "" {
			t.Errorf("知识点缺少 code/name: %+v", point)
		}
		if _, ok := seen[point.Code]; ok {
			t.Errorf("知识点 code 重复: %s", point.Code)
		}
		seen[point.Code] = struct{}{}
		if len(point.Keywords) == 0 {
			t.Errorf("知识点 %s 没有关键词，无法关联资源", point.Code)
		}
	}
}

func TestDefaultKnowledgeEdgesReferenceExistingPoints(t *testing.T) {
	codes := map[string]struct{}{}
	for _, point := range service.DefaultKnowledgePoints {
		codes[point.Code] = struct{}{}
	}
	for _, edge := range service.DefaultKnowledgeEdges {
		if _, ok := codes[edge.From]; !ok {
			t.Errorf("先修关系引用不存在的知识点: %s", edge.From)
		}
		if _, ok := codes[edge.To]; !ok {
			t.Errorf("先修关系引用不存在的知识点: %s", edge.To)
		}
	}
}

func TestMatchScore(t *testing.T) {
	keywords := []string{"机器学习"}

	if score := service.MatchScore("机器学习入门到实战", nil, "", keywords); score != 1 {
		t.Errorf("标题命中 score = %v, want 1", score)
	}
	if score := service.MatchScore("某课程", []string{"机器学习", "AI"}, "", keywords); score < 0.5 {
		t.Errorf("标签命中 score = %v, want >= 0.5", score)
	}
	if score := service.MatchScore("某课程", nil, "本课程介绍机器学习", keywords); score >= 0.5 {
		t.Errorf("仅描述命中 score = %v, 应低于 0.5 阈值", score)
	}
	if score := service.MatchScore("数学分析", []string{"高等数学"}, "微积分", keywords); score != 0 {
		t.Errorf("未命中 score = %v, want 0", score)
	}
	if score := service.MatchScore("机器学习", []string{"机器学习"}, "机器学习", keywords); score != 1 {
		t.Errorf("多字段命中应封顶为 1，实际 %v", score)
	}
}
