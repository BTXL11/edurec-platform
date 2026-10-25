package service_test

import (
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

func brief(id uint, code string, level int) service.PointBrief {
	return service.PointBrief{ID: id, Code: code, Name: code, Level: level}
}

func TestPrerequisiteOrderChain(t *testing.T) {
	points := []service.PointBrief{
		brief(1, "calculus", 1),
		brief(2, "analysis", 2),
		brief(3, "functional", 3),
	}
	edges := []service.KnowledgeEdgeSeed{
		{From: "calculus", To: "analysis"},
		{From: "analysis", To: "functional"},
	}

	ordered, err := service.PrerequisiteOrder("functional", points, edges)
	if err != nil {
		t.Fatalf("PrerequisiteOrder() error = %v", err)
	}

	want := []string{"calculus", "analysis", "functional"}
	if len(ordered) != len(want) {
		t.Fatalf("len = %d, want %d", len(ordered), len(want))
	}
	for i := range want {
		if ordered[i].Code != want[i] {
			t.Errorf("ordered[%d] = %s, want %s", i, ordered[i].Code, want[i])
		}
	}
}

// 目标只需要自己的先修闭包，无关节点不应进入路径
func TestPrerequisiteOrderOnlyClosure(t *testing.T) {
	points := []service.PointBrief{
		brief(1, "calculus", 1),
		brief(2, "analysis", 2),
		brief(3, "linear", 1),
		brief(4, "ml", 3),
	}
	edges := []service.KnowledgeEdgeSeed{
		{From: "calculus", To: "analysis"},
		{From: "linear", To: "ml"},
	}

	ordered, err := service.PrerequisiteOrder("analysis", points, edges)
	if err != nil {
		t.Fatalf("PrerequisiteOrder() error = %v", err)
	}
	if len(ordered) != 2 || ordered[0].Code != "calculus" || ordered[1].Code != "analysis" {
		t.Fatalf("ordered = %v, want [calculus analysis]", ordered)
	}
}

func TestPrerequisiteOrderRejectsUnknownTarget(t *testing.T) {
	_, err := service.PrerequisiteOrder("nope", []service.PointBrief{brief(1, "a", 1)}, nil)
	if err == nil {
		t.Fatal("未知目标应报错")
	}
}

func TestPrerequisiteOrderRejectsCycle(t *testing.T) {
	points := []service.PointBrief{brief(1, "a", 1), brief(2, "b", 1)}
	edges := []service.KnowledgeEdgeSeed{
		{From: "a", To: "b"},
		{From: "b", To: "a"},
	}

	if _, err := service.PrerequisiteOrder("b", points, edges); err == nil {
		t.Fatal("有环时应报错")
	}
}

func TestBuildPathStepsMarksDoneCurrentLocked(t *testing.T) {
	ordered := []service.PointBrief{
		brief(1, "calculus", 1),
		brief(2, "analysis", 2),
		brief(3, "functional", 3),
	}
	mastery := map[string]float32{"calculus": 1, "analysis": 0.2, "functional": 0}
	counts := map[string]int64{"calculus": 10, "analysis": 5, "functional": 2}

	steps := service.BuildPathSteps(ordered, mastery, counts)

	if steps[0].Status != service.StepDone {
		t.Errorf("已掌握节点应为 done，got %s", steps[0].Status)
	}
	if steps[1].Status != service.StepCurrent {
		t.Errorf("第一个未掌握节点应为 current，got %s", steps[1].Status)
	}
	if steps[2].Status != service.StepLocked {
		t.Errorf("后续节点应为 locked，got %s", steps[2].Status)
	}
	if steps[0].ResourceCount != 10 {
		t.Errorf("ResourceCount = %d, want 10", steps[0].ResourceCount)
	}
}

// 问卷自评高（>=0.8）也算已掌握，不再作为待学步骤
func TestBuildPathStepsSkipsMasteredNodes(t *testing.T) {
	ordered := []service.PointBrief{brief(1, "a", 1), brief(2, "b", 2)}
	steps := service.BuildPathSteps(ordered, map[string]float32{"a": 0.9, "b": 0.1}, nil)

	if steps[0].Status != service.StepDone {
		t.Errorf("a 应为 done，got %s", steps[0].Status)
	}
	if steps[1].Status != service.StepCurrent {
		t.Errorf("b 应为 current，got %s", steps[1].Status)
	}
}

func TestAdvancePathUnlocksNext(t *testing.T) {
	steps := []model.PathStep{
		{Code: "a", Status: service.StepDone},
		{Code: "b", Status: service.StepCurrent},
		{Code: "c", Status: service.StepLocked},
	}

	updated, index, status, err := service.AdvancePath(steps, 1)
	if err != nil {
		t.Fatalf("AdvancePath() error = %v", err)
	}
	if updated[1].Status != service.StepDone || updated[1].Mastery != 1 {
		t.Errorf("完成步骤应置 done 且掌握度为 1")
	}
	if updated[2].Status != service.StepCurrent {
		t.Errorf("下一节应解锁为 current，got %s", updated[2].Status)
	}
	if index != 2 || status != "active" {
		t.Errorf("index=%d status=%s, want 2 active", index, status)
	}
}

func TestAdvancePathCompletesPath(t *testing.T) {
	steps := []model.PathStep{
		{Code: "a", Status: service.StepDone},
		{Code: "b", Status: service.StepCurrent},
	}

	_, _, status, err := service.AdvancePath(steps, 1)
	if err != nil {
		t.Fatalf("AdvancePath() error = %v", err)
	}
	if status != service.PathCompleted {
		t.Errorf("status = %s, want %s", status, service.PathCompleted)
	}
}

func TestAdvancePathRejectsLockedStep(t *testing.T) {
	steps := []model.PathStep{
		{Code: "a", Status: service.StepCurrent},
		{Code: "b", Status: service.StepLocked},
	}
	if _, _, _, err := service.AdvancePath(steps, 1); err == nil {
		t.Fatal("未解锁步骤应被拒绝")
	}
}
