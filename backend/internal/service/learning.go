package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// LearningService 自适应学习：问卷自评 -> 知识画像 -> 先修路径 -> 逐节解锁。
type LearningService struct {
	knowledge repository.KnowledgeRepository
	learning  repository.LearningRepository
}

func NewLearningService(
	knowledge repository.KnowledgeRepository,
	learning repository.LearningRepository,
) *LearningService {
	return &LearningService{knowledge: knowledge, learning: learning}
}

// ProfileItem 知识画像条目：某知识点的掌握度与来源
type ProfileItem struct {
	PointCode string  `json:"point_code"`
	Name      string  `json:"name"`
	Level     int     `json:"level"`
	Mastery   float32 `json:"mastery"`
	Source    string  `json:"source"`
}

// LearningPathView 学习路径视图
type LearningPathView struct {
	ID           uint             `json:"id"`
	TargetCode   string           `json:"target_code"`
	TargetName   string           `json:"target_name"`
	Steps        []model.PathStep `json:"steps"`
	CurrentIndex int              `json:"current_index"`
	Status       string           `json:"status"`
}

// Profile 返回用户对全部知识点的掌握情况。
//
// 掌握度 = max(画像里存储的值, 行为侧估计值)。行为只给到 0.6 上限：
// 仅靠看视频不应自动判定「已掌握」，达到 0.8 阈值仍需问卷自评或完成学习路径。
func (s *LearningService) Profile(userID uint) ([]ProfileItem, error) {
	points, err := s.knowledge.ListPoints()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	profiles, err := s.learning.ListProfiles(userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	byCode := make(map[string]model.LearnerProfile, len(profiles))
	for _, profile := range profiles {
		byCode[profile.PointCode] = profile
	}

	engagement, err := s.learning.UserEngagementByPoint(userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	behaviorMastery := make(map[uint]float32, len(engagement))
	for pointID, count := range engagement {
		value := float32(count) * 0.1
		if value > 0.6 {
			value = 0.6
		}
		behaviorMastery[pointID] = value
	}

	items := make([]ProfileItem, 0, len(points))
	for i := range points {
		point := &points[i]
		mastery := float32(0)
		source := ""
		if profile, ok := byCode[point.Code]; ok {
			mastery = profile.Mastery
			source = profile.Source
		}
		if value := behaviorMastery[point.ID]; value > mastery {
			mastery = value
			source = "behavior"
		}
		items = append(items, ProfileItem{
			PointCode: point.Code,
			Name:      point.Name,
			Level:     point.Level,
			Mastery:   mastery,
			Source:    source,
		})
	}
	return items, nil
}

// SubmitQuestionnaire 保存初始问卷，把自评等级（0-3）换算成掌握度写入画像。
func (s *LearningService) SubmitQuestionnaire(userID uint, answers map[string]int) ([]ProfileItem, error) {
	points, err := s.knowledge.ListPoints()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	valid := make(map[string]struct{}, len(points))
	for i := range points {
		valid[points[i].Code] = struct{}{}
	}

	cleaned := make(map[string]int, len(answers))
	for code, level := range answers {
		code = strings.TrimSpace(code)
		if _, ok := valid[code]; !ok {
			return nil, apperror.BadRequest(fmt.Sprintf("未知知识点: %s", code))
		}
		if level < 0 || level > 3 {
			return nil, apperror.BadRequest("自评等级必须在 0-3 之间")
		}
		cleaned[code] = level
	}

	encoded, err := json.Marshal(cleaned)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.learning.SaveQuestionnaire(userID, string(encoded)); err != nil {
		return nil, apperror.Internal(err)
	}

	for code, level := range cleaned {
		if err := s.learning.UpsertProfile(&model.LearnerProfile{
			UserID:    userID,
			PointCode: code,
			Mastery:   float32(level) / 3,
			Source:    "questionnaire",
		}); err != nil {
			return nil, apperror.Internal(err)
		}
	}
	return s.Profile(userID)
}

// BuildPath 为「目标知识点」生成学习路径：
// 先取先修闭包并拓扑排序，再按画像跳过已掌握节点，只留未掌握内容逐节解锁。
func (s *LearningService) BuildPath(userID uint, targetCode string) (*LearningPathView, error) {
	targetCode = strings.TrimSpace(targetCode)
	if targetCode == "" {
		return nil, apperror.BadRequest("目标知识点不能为空")
	}

	points, err := s.knowledge.ListPoints()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	briefs := make([]PointBrief, 0, len(points))
	codeByID := make(map[uint]string, len(points))
	nameByCode := make(map[string]string, len(points))
	levelByCode := make(map[string]int, len(points))
	for i := range points {
		briefs = append(briefs, PointBrief{
			ID: points[i].ID, Code: points[i].Code, Name: points[i].Name, Level: points[i].Level,
		})
		codeByID[points[i].ID] = points[i].Code
		nameByCode[points[i].Code] = points[i].Name
		levelByCode[points[i].Code] = points[i].Level
	}

	edges, err := s.knowledge.ListEdges()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	edgeSeeds := make([]KnowledgeEdgeSeed, 0, len(edges))
	for i := range edges {
		from, okFrom := codeByID[edges[i].FromID]
		to, okTo := codeByID[edges[i].ToID]
		if !okFrom || !okTo {
			continue
		}
		edgeSeeds = append(edgeSeeds, KnowledgeEdgeSeed{From: from, To: to})
	}

	ordered, err := PrerequisiteOrder(targetCode, briefs, edgeSeeds)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	mastery, err := s.masteryByCode(userID, points)
	if err != nil {
		return nil, err
	}

	countByID, err := s.knowledge.CountResourcesByPoint()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	resourceCounts := make(map[string]int64, len(countByID))
	for id, count := range countByID {
		if code, ok := codeByID[id]; ok {
			resourceCounts[code] = count
		}
	}

	steps := BuildPathSteps(ordered, mastery, resourceCounts)
	status := "active"
	currentIndex := 0
	for i := range steps {
		if steps[i].Status == StepCurrent {
			currentIndex = i
			break
		}
	}
	if allDone(steps) {
		status = PathCompleted
		currentIndex = len(steps) - 1
	}

	encoded, err := json.Marshal(steps)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if err := s.learning.ArchiveActivePaths(userID, targetCode); err != nil {
		return nil, apperror.Internal(err)
	}
	path := &model.LearningPath{
		UserID:       userID,
		TargetCode:   targetCode,
		Steps:        string(encoded),
		CurrentIndex: currentIndex,
		Status:       status,
	}
	if err := s.learning.CreatePath(path); err != nil {
		return nil, apperror.Internal(err)
	}

	return &LearningPathView{
		ID:           path.ID,
		TargetCode:   path.TargetCode,
		TargetName:   nameByCode[targetCode],
		Steps:        steps,
		CurrentIndex: path.CurrentIndex,
		Status:       path.Status,
	}, nil
}

// GetPath 读取路径并校验归属
func (s *LearningService) GetPath(userID, pathID uint) (*LearningPathView, error) {
	path, err := s.learning.FindPathByID(pathID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if path == nil || path.UserID != userID {
		return nil, apperror.NotFound("学习路径不存在")
	}
	return s.toPathView(path)
}

// CompleteStep 完成当前步骤并解锁下一节；跳步（完成 locked）会被拒绝。
func (s *LearningService) CompleteStep(userID, pathID uint, index int) (*LearningPathView, error) {
	path, err := s.learning.FindPathByID(pathID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if path == nil || path.UserID != userID {
		return nil, apperror.NotFound("学习路径不存在")
	}
	if path.Status == PathCompleted {
		return s.toPathView(path)
	}

	var steps []model.PathStep
	if err := json.Unmarshal([]byte(path.Steps), &steps); err != nil {
		return nil, apperror.Internal(err)
	}

	updated, currentIndex, status, err := AdvancePath(steps, index)
	if err != nil {
		return nil, apperror.BadRequest(err.Error())
	}

	if err := s.learning.UpsertProfile(&model.LearnerProfile{
		UserID:    userID,
		PointCode: steps[index].Code,
		Mastery:   1,
		Source:    "complete",
	}); err != nil {
		return nil, apperror.Internal(err)
	}

	encoded, err := json.Marshal(updated)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	path.Steps = string(encoded)
	path.CurrentIndex = currentIndex
	path.Status = status
	if err := s.learning.UpdatePath(path); err != nil {
		return nil, apperror.Internal(err)
	}
	return s.toPathView(path)
}

// PathResources 返回路径当前步骤对应的课程资源
func (s *LearningService) PathResources(userID, pathID uint, limit int) (*model.KnowledgePoint, []model.Resource, error) {
	view, err := s.GetPath(userID, pathID)
	if err != nil {
		return nil, nil, err
	}
	if len(view.Steps) == 0 {
		return nil, nil, apperror.BadRequest("学习路径为空")
	}
	current := view.Steps[view.CurrentIndex]
	point, err := s.knowledge.FindPointByCode(current.Code)
	if err != nil {
		return nil, nil, apperror.Internal(err)
	}
	if point == nil {
		return nil, nil, apperror.NotFound("知识点不存在")
	}
	items, _, err := s.knowledge.ListResourcesByPoint(point.ID, limit, 0)
	if err != nil {
		return nil, nil, apperror.Internal(err)
	}
	return point, items, nil
}

// masteryByCode 合并「画像存储值」与「行为估计值」，返回每个知识点的掌握度
func (s *LearningService) masteryByCode(userID uint, points []model.KnowledgePoint) (map[string]float32, error) {
	profiles, err := s.learning.ListProfiles(userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	out := make(map[string]float32, len(profiles))
	for _, profile := range profiles {
		out[profile.PointCode] = profile.Mastery
	}

	engagement, err := s.learning.UserEngagementByPoint(userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	idToCode := make(map[uint]string, len(points))
	for i := range points {
		idToCode[points[i].ID] = points[i].Code
	}
	for id, count := range engagement {
		code, ok := idToCode[id]
		if !ok {
			continue
		}
		value := float32(count) * 0.1
		if value > 0.6 {
			value = 0.6
		}
		if value > out[code] {
			out[code] = value
		}
	}
	return out, nil
}

func allDone(steps []model.PathStep) bool {
	for i := range steps {
		if steps[i].Status != StepDone {
			return false
		}
	}
	return len(steps) > 0
}

func (s *LearningService) toPathView(path *model.LearningPath) (*LearningPathView, error) {
	var steps []model.PathStep
	if err := json.Unmarshal([]byte(path.Steps), &steps); err != nil {
		return nil, apperror.Internal(err)
	}
	name := path.TargetCode
	if point, err := s.knowledge.FindPointByCode(path.TargetCode); err == nil && point != nil {
		name = point.Name
	}
	return &LearningPathView{
		ID:           path.ID,
		TargetCode:   path.TargetCode,
		TargetName:   name,
		Steps:        steps,
		CurrentIndex: path.CurrentIndex,
		Status:       path.Status,
	}, nil
}

