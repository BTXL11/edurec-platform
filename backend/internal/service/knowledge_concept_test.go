package service_test

import (
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

// 课程名 -> 学科：MOOCCube 的课程名带「（自主模式）」「（2019春）」等后缀，
// 仍应稳定归到学科。
func TestMatchSubjectCode(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"线性代数（自主模式）", "linear-algebra"},
		{"线性代数(2)（自主模式）", "linear-algebra"},
		{"高等数学（自主模式）", "calculus"},
		{"高等数学习题课——级数，多元函数微积分（2019春）", "calculus"},
		{"概率论与数理统计（2019春）", "probability"},
		{"大数据机器学习（2019春）", "machine-learning"},
		{"深度学习基础（2020秋）", "deep-learning"},
		{"数据结构与算法（自主模式）", "data-structures"},
		{"雅思写作单项班", "ielts"},
		{"大学英语（自主模式）", "college-english"},
		{"", ""},
	}

	for _, tc := range cases {
		if got := service.MatchSubjectCode(tc.name); got != tc.want {
			t.Errorf("MatchSubjectCode(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// 概念 -> 学科：由「课程 -> 概念」投票决定；票数相同按学科定义顺序取，保证可复现。
func TestAssignConceptSubjects(t *testing.T) {
	courseNames := map[string]string{
		"c1": "线性代数（自主模式）",
		"c2": "概率论与数理统计（2019春）",
		"c3": "没有学科关键词的课程",
	}
	courseConcepts := map[string][]string{
		"c1": {"K_A", "K_B"},
		"c2": {"K_B", "K_C"},
		"c3": {"K_A"},
	}

	got := service.AssignConceptSubjects(
		[]string{"K_A", "K_B", "K_C", "K_UNKNOWN"},
		courseConcepts,
		courseNames,
	)

	if got["K_A"] != "linear-algebra" {
		t.Errorf("K_A = %q, want linear-algebra", got["K_A"])
	}
	// K_B 在线性代数与概率论各一票，按学科定义顺序取线性代数
	if got["K_B"] != "linear-algebra" {
		t.Errorf("K_B = %q, want linear-algebra（平票按学科顺序）", got["K_B"])
	}
	if got["K_C"] != "probability" {
		t.Errorf("K_C = %q, want probability", got["K_C"])
	}
	if _, ok := got["K_UNKNOWN"]; ok {
		t.Error("没有课程包含的概念不应被分配学科")
	}
}

// 归属只来自能匹配学科的课程：全部课程都无关键词时不产生任何分配
func TestAssignConceptSubjectsWithoutSubjectCourses(t *testing.T) {
	got := service.AssignConceptSubjects(
		[]string{"K_A"},
		map[string][]string{"c1": {"K_A"}},
		map[string]string{"c1": "某通识课程"},
	)
	if len(got) != 0 {
		t.Fatalf("got %v, want 空", got)
	}
}
