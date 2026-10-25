package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// CrawlStatus 学科资源自动爬取任务状态
type CrawlStatus struct {
	DisciplineCode string     `json:"discipline_code"`
	Status         string     `json:"status"` // idle / running / done / failed
	Imported       int        `json:"imported"`
	Linked         int        `json:"linked"`
	ResourceTotal  int64      `json:"resource_total"`
	Keyword        string     `json:"keyword,omitempty"`
	Error          string     `json:"error,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
}

// DisciplineCrawlService 学科无资源时自动去 B 站搜索并挂到学科知识点下。
//
// 只在学科资源数为 0 时触发（避免每次打开页面都重复爬取），有并发去重与冷却。
type DisciplineCrawlService struct {
	repo     repository.KnowledgeRepository
	online   *BilibiliOnlineService
	cooldown time.Duration

	mu   sync.Mutex
	jobs map[string]*CrawlStatus
}

func NewDisciplineCrawlService(
	repo repository.KnowledgeRepository,
	online *BilibiliOnlineService,
	cooldown time.Duration,
) *DisciplineCrawlService {
	if cooldown <= 0 {
		cooldown = 30 * time.Minute
	}
	return &DisciplineCrawlService{
		repo:     repo,
		online:   online,
		cooldown: cooldown,
		jobs:     make(map[string]*CrawlStatus),
	}
}

// Ensure 在学科没有资源时启动一次自动爬取；已有资源、正在跑或冷却期内直接返回状态。
func (s *DisciplineCrawlService) Ensure(code string) (*CrawlStatus, error) {
	code = strings.TrimSpace(code)
	discipline, err := s.repo.FindDisciplineByCode(code)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if discipline == nil {
		return nil, apperror.NotFound("学科不存在")
	}

	s.mu.Lock()
	if current, ok := s.jobs[code]; ok {
		if current.Status == "running" ||
			(current.Status == "done" && current.FinishedAt != nil && time.Since(*current.FinishedAt) < s.cooldown) {
			snapshot := *current
			s.mu.Unlock()
			return &snapshot, nil
		}
	}
	started := time.Now()
	status := &CrawlStatus{DisciplineCode: code, Status: "running", StartedAt: &started}
	s.jobs[code] = status
	snapshot := *status
	s.mu.Unlock()

	go s.run(discipline)
	return &snapshot, nil
}

// Status 查询爬取状态；没有任务时返回 idle（带上当前资源数）。
func (s *DisciplineCrawlService) Status(code string) (*CrawlStatus, error) {
	code = strings.TrimSpace(code)
	discipline, err := s.repo.FindDisciplineByCode(code)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if discipline == nil {
		return nil, apperror.NotFound("学科不存在")
	}

	s.mu.Lock()
	if current, ok := s.jobs[code]; ok {
		snapshot := *current
		s.mu.Unlock()
		return &snapshot, nil
	}
	s.mu.Unlock()

	counts, err := s.repo.CountResourcesByDiscipline()
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return &CrawlStatus{DisciplineCode: code, Status: "idle", ResourceTotal: counts[code]}, nil
}

func (s *DisciplineCrawlService) run(discipline *model.Discipline) {
	status := s.statusOf(discipline.Code)
	defer func() {
		finished := time.Now()
		s.mu.Lock()
		status.FinishedAt = &finished
		s.mu.Unlock()
	}()

	// 已有资源就不重复爬
	counts, err := s.repo.CountResourcesByDiscipline()
	if err != nil {
		s.fail(status, err)
		return
	}
	if counts[discipline.Code] > 0 {
		s.mu.Lock()
		status.Status = "done"
		status.ResourceTotal = counts[discipline.Code]
		s.mu.Unlock()
		return
	}

	keywords := disciplineKeywords(discipline)
	if len(keywords) == 0 {
		keywords = []string{discipline.Name}
	}
	if len(keywords) > 2 {
		keywords = keywords[:2]
	}

	points, err := s.repo.ListPointsByDiscipline(discipline.Code)
	if err != nil {
		s.fail(status, err)
		return
	}

	importedTotal := 0
	linkedTotal := 0
	var lastErr error
	for _, keyword := range keywords {
		s.mu.Lock()
		status.Keyword = keyword
		s.mu.Unlock()

		resources, _, err := s.online.SearchAndImport(keyword, 1)
		if err != nil {
			lastErr = err
			continue
		}
		importedTotal += len(resources)
		linkedTotal += s.linkResources(points, resources)
	}

	s.mu.Lock()
	status.Imported = importedTotal
	status.Linked = linkedTotal
	s.mu.Unlock()

	if importedTotal == 0 && lastErr != nil {
		s.fail(status, lastErr)
		return
	}

	updatedCounts, err := s.repo.CountResourcesByDiscipline()
	if err != nil {
		s.fail(status, err)
		return
	}
	s.mu.Lock()
	status.Status = "done"
	status.ResourceTotal = updatedCounts[discipline.Code]
	s.mu.Unlock()
}

// linkResources 按知识点关键词给新资源打分并建立关联
func (s *DisciplineCrawlService) linkResources(points []model.KnowledgePoint, resources []model.Resource) int {
	keywordsByCode := make(map[string][]string, len(points))
	for i := range points {
		keywordsByCode[points[i].Code] = pointKeywords(points[i])
	}

	linked := 0
	for i := range resources {
		resource := &resources[i]
		tags := resourceTagList(resource.Tags)
		for j := range points {
			point := &points[j]
			score := MatchScore(resource.Title, tags, resource.Description, keywordsByCode[point.Code])
			if score < 0.5 {
				continue
			}
			if err := s.repo.UpsertResourceLink(&model.ResourceKnowledgePoint{
				ResourceID: resource.ID,
				PointID:    point.ID,
				Weight:     float32(score),
			}); err != nil {
				continue
			}
			linked++
		}
	}
	return linked
}

func (s *DisciplineCrawlService) statusOf(code string) *CrawlStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.jobs[code]
}

func (s *DisciplineCrawlService) fail(status *CrawlStatus, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	status.Status = "failed"
	status.Error = fmt.Sprintf("%v", err)
}

// disciplineKeywords 解析学科关键词 JSON；空或损坏时退回学科名
func disciplineKeywords(discipline *model.Discipline) []string {
	var keywords []string
	if err := json.Unmarshal([]byte(discipline.Keywords), &keywords); err != nil {
		return nil
	}
	out := make([]string, 0, len(keywords))
	for _, keyword := range keywords {
		if trimmed := strings.TrimSpace(keyword); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// pointKeywords 取知识点种子里的关键词；没有则退回名称
func pointKeywords(point model.KnowledgePoint) []string {
	for _, seed := range DefaultKnowledgePoints {
		if seed.Code == point.Code && len(seed.Keywords) > 0 {
			return seed.Keywords
		}
	}
	return []string{point.Name}
}

func resourceTagList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil
	}
	return tags
}
