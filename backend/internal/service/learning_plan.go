package service

import (
	"fmt"
	"sort"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
)

// PointBrief 学习路径计算所需的最小知识点信息
type PointBrief struct {
	ID    uint
	Code  string
	Name  string
	Level int
}

// 路径步骤状态
const (
	StepDone    = "done"
	StepCurrent = "current"
	StepLocked  = "locked"
	// PathCompleted 整条路径已学完
	PathCompleted = "completed"
)

// PrerequisiteOrder 返回目标知识点的「先修闭包 + 自身」，按学习顺序排列。
//
// 顺序由先修边做拓扑排序决定（同层按 level、ID 排序保证稳定）；
// 目标不存在或闭包内有环时返回错误。缺失一端的边直接忽略（图谱可能尚未重建）。
func PrerequisiteOrder(target string, points []PointBrief, edges []KnowledgeEdgeSeed) ([]PointBrief, error) {
	byCode := make(map[string]PointBrief, len(points))
	for _, point := range points {
		byCode[point.Code] = point
	}
	if _, ok := byCode[target]; !ok {
		return nil, fmt.Errorf("知识点不存在: %s", target)
	}

	// 反向边：to -> 它的先修集合
	prereqs := make(map[string][]string, len(edges))
	for _, edge := range edges {
		if _, okFrom := byCode[edge.From]; !okFrom {
			continue
		}
		if _, okTo := byCode[edge.To]; !okTo {
			continue
		}
		prereqs[edge.To] = append(prereqs[edge.To], edge.From)
	}

	// 先修闭包（含目标自身）
	closure := make(map[string]struct{})
	stack := []string{target}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, seen := closure[current]; seen {
			continue
		}
		closure[current] = struct{}{}
		stack = append(stack, prereqs[current]...)
	}

	// 闭包内做拓扑排序
	inDegree := make(map[string]int, len(closure))
	next := make(map[string][]string, len(closure))
	for code := range closure {
		inDegree[code] = 0
	}
	for _, edge := range edges {
		if _, ok := closure[edge.From]; !ok {
			continue
		}
		if _, ok := closure[edge.To]; !ok {
			continue
		}
		next[edge.From] = append(next[edge.From], edge.To)
		inDegree[edge.To]++
	}

	queue := make([]string, 0, len(closure))
	for code, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, code)
		}
	}
	sortByLevelID(queue, byCode)

	ordered := make([]PointBrief, 0, len(closure))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		ordered = append(ordered, byCode[current])

		added := make([]string, 0)
		for _, to := range next[current] {
			inDegree[to]--
			if inDegree[to] == 0 {
				added = append(added, to)
			}
		}
		sortByLevelID(added, byCode)
		queue = append(queue, added...)
		// 队列整体保持有序，便于同层稳定输出
		sortByLevelID(queue, byCode)
	}

	if len(ordered) != len(closure) {
		return nil, fmt.Errorf("知识点先修关系存在环，无法生成学习路径")
	}
	return ordered, nil
}

func sortByLevelID(codes []string, byCode map[string]PointBrief) {
	sort.SliceStable(codes, func(i, j int) bool {
		left, right := byCode[codes[i]], byCode[codes[j]]
		if left.Level != right.Level {
			return left.Level < right.Level
		}
		return left.ID < right.ID
	})
}

// BuildPathSteps 根据已掌握情况把有序知识点转成学习路径步骤：
//
//   - 掌握度 >= 阈值：done（已在画像里，不需要重学，但仍保留在路径中作为依据）；
//   - 第一个未掌握：current；
//   - 其余：locked（学完前一节才解锁）。
//
// 全部已掌握时所有步骤 done，调用方把路径状态置为 completed。
func BuildPathSteps(
	ordered []PointBrief,
	mastery map[string]float32,
	resourceCounts map[string]int64,
) []model.PathStep {
	steps := make([]model.PathStep, 0, len(ordered))
	currentAssigned := false

	for _, point := range ordered {
		value := mastery[point.Code]
		status := StepLocked
		switch {
		case value >= model.MasteryThreshold:
			status = StepDone
		case !currentAssigned:
			status = StepCurrent
			currentAssigned = true
		}
		steps = append(steps, model.PathStep{
			Code:          point.Code,
			Name:          point.Name,
			Level:         point.Level,
			Mastery:       value,
			Status:        status,
			ResourceCount: resourceCounts[point.Code],
		})
	}

	// 若没有未掌握的节点（全部 done），current 不存在，调用方据 done 数判断完成
	return steps
}

// AdvancePath 完成当前步骤并解锁下一节。
//
// 只允许完成 status=current 的步骤；locked 步骤返回错误（前端不应能跳步）。
// 返回更新后的步骤、新的 CurrentIndex 与路径状态（active / completed）。
func AdvancePath(steps []model.PathStep, index int) ([]model.PathStep, int, string, error) {
	if index < 0 || index >= len(steps) {
		return nil, 0, "", fmt.Errorf("步骤不存在: %d", index)
	}
	switch steps[index].Status {
	case StepLocked:
		return nil, 0, "", fmt.Errorf("该步骤尚未解锁")
	case StepDone:
		// 幂等：重复完成同一节不算错误，返回当前状态
	default:
		steps[index].Status = StepDone
		steps[index].Mastery = 1
	}

	for i := range steps {
		if steps[i].Status == StepCurrent {
			return steps, i, "active", nil
		}
	}
	// 没有 current：找第一个未完成的步骤并解锁；都完成则路径结束
	for i := range steps {
		if steps[i].Status != StepDone {
			steps[i].Status = StepCurrent
			return steps, i, "active", nil
		}
	}
	return steps, len(steps) - 1, PathCompleted, nil
}
