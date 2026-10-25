package service_test

import (
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

// 学科种子必须覆盖主要学科，且每个知识点都挂在存在的学科下
func TestDefaultDisciplinesConsistent(t *testing.T) {
	disciplines := make(map[string]struct{}, len(service.DefaultDisciplines))
	for _, discipline := range service.DefaultDisciplines {
		if discipline.Code == "" || discipline.Name == "" {
			t.Errorf("学科缺少 code/name: %+v", discipline)
		}
		if _, exists := disciplines[discipline.Code]; exists {
			t.Errorf("学科 code 重复: %s", discipline.Code)
		}
		disciplines[discipline.Code] = struct{}{}
		if len(discipline.Keywords) == 0 {
			t.Errorf("学科 %s 缺少关键词（自动爬取需要）", discipline.Code)
		}
	}

	for _, point := range service.DefaultKnowledgePoints {
		if point.DisciplineCode == "" {
			t.Errorf("知识点 %s 未归入学科", point.Code)
			continue
		}
		if _, ok := disciplines[point.DisciplineCode]; !ok {
			t.Errorf("知识点 %s 引用了不存在的学科 %s", point.Code, point.DisciplineCode)
		}
	}

	if len(disciplines) < 15 {
		t.Errorf("主要学科数量过少: %d", len(disciplines))
	}
}
