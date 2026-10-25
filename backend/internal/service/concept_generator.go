package service

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// GeneratedConcept engine 生成的一个概念
type GeneratedConcept struct {
	Name     string  `json:"name"`
	PageRank float64 `json:"page_rank"`
}

// ConceptGenerationRunner 概念生成能力提供方（真实实现是 engine 的 HTTP 服务）
type ConceptGenerationRunner interface {
	Generate(ctx context.Context, point *model.KnowledgePoint) ([]GeneratedConcept, error)
}

// HTTPConceptRunner 调用 engine 侧 kg_server.py 生成概念。
//
// 平台不直接执行 Python：算法与 venv 依赖都在 engine，平台只发 HTTP 请求。
type HTTPConceptRunner struct {
	baseURL string
	client  *http.Client
}

func NewHTTPConceptRunner(baseURL string, timeout time.Duration) *HTTPConceptRunner {
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	return &HTTPConceptRunner{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

type generateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PointCode   string `json:"point_code"`
}

type generateResponse struct {
	Concepts []GeneratedConcept `json:"concepts"`
	Error    string             `json:"error"`
}

func (r *HTTPConceptRunner) Generate(ctx context.Context, point *model.KnowledgePoint) ([]GeneratedConcept, error) {
	payload, err := json.Marshal(generateRequest{
		Title:       point.Name,
		Description: point.Description,
		PointCode:   point.Code,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		r.baseURL+"/knowledge/generate", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("engine 知识生成服务不可用: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("读取 engine 响应失败: %w", err)
	}
	var parsed generateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("engine 响应不是合法 JSON: %w", err)
	}
	if resp.StatusCode >= 400 {
		message := parsed.Error
		if message == "" {
			message = strings.TrimSpace(string(body))
		}
		return nil, fmt.Errorf("engine 生成失败（HTTP %d）: %s", resp.StatusCode, message)
	}
	return parsed.Concepts, nil
}

// GenerationStatus 一次概念生成任务的状态
type GenerationStatus struct {
	PointCode  string     `json:"point_code"`
	Status     string     `json:"status"` // idle / running / done / failed
	Generated  int        `json:"generated"`
	Total      int        `json:"total"`
	Error      string     `json:"error,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

// ConceptGenerator 管理「按学科生成概念」的异步任务：
// 搜索时触发一次，前端轮询状态；同一学科并发触发会复用同一个任务。
type ConceptGenerator struct {
	knowledge repository.KnowledgeRepository
	runner    ConceptGenerationRunner
	cooldown  time.Duration

	mu   sync.Mutex
	jobs map[string]*GenerationStatus
}

func NewConceptGenerator(
	knowledge repository.KnowledgeRepository,
	runner ConceptGenerationRunner,
	cooldown time.Duration,
) *ConceptGenerator {
	if cooldown <= 0 {
		cooldown = 10 * time.Minute
	}
	return &ConceptGenerator{
		knowledge: knowledge,
		runner:    runner,
		cooldown:  cooldown,
		jobs:      make(map[string]*GenerationStatus),
	}
}

// Start 异步触发一次生成；正在跑或刚跑过（冷却期内）时直接返回现有状态。
func (g *ConceptGenerator) Start(code string) (*GenerationStatus, error) {
	point, err := g.knowledge.FindPointByCode(strings.TrimSpace(code))
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if point == nil {
		return nil, apperror.NotFound("知识点不存在")
	}

	g.mu.Lock()
	if current, ok := g.jobs[point.Code]; ok {
		if current.Status == "running" || (current.Status == "done" && current.FinishedAt != nil &&
			time.Since(*current.FinishedAt) < g.cooldown) {
			snapshot := *current
			g.mu.Unlock()
			return &snapshot, nil
		}
	}
	started := time.Now()
	status := &GenerationStatus{PointCode: point.Code, Status: "running", StartedAt: &started}
	g.jobs[point.Code] = status
	snapshot := *status
	g.mu.Unlock()

	go g.run(point)

	return &snapshot, nil
}

// Status 查询生成状态；没有任务时返回 idle（并带上当前概念总数）。
func (g *ConceptGenerator) Status(code string) (*GenerationStatus, error) {
	point, err := g.knowledge.FindPointByCode(strings.TrimSpace(code))
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if point == nil {
		return nil, apperror.NotFound("知识点不存在")
	}

	g.mu.Lock()
	if current, ok := g.jobs[point.Code]; ok {
		snapshot := *current
		g.mu.Unlock()
		return &snapshot, nil
	}
	g.mu.Unlock()

	total, err := g.totalConcepts(point.Code)
	if err != nil {
		return nil, apperror.Internal(err)
	}
	return &GenerationStatus{PointCode: point.Code, Status: "idle", Total: total}, nil
}

// GenerateNow 同步生成（供批量重建命令使用），不受冷却时间限制。
func (g *ConceptGenerator) GenerateNow(ctx context.Context, code string) (*GenerationStatus, error) {
	point, err := g.knowledge.FindPointByCode(strings.TrimSpace(code))
	if err != nil {
		return nil, apperror.Internal(err)
	}
	if point == nil {
		return nil, apperror.NotFound("知识点不存在")
	}

	g.mu.Lock()
	if current, ok := g.jobs[point.Code]; ok && current.Status == "running" {
		g.mu.Unlock()
		return nil, apperror.BadRequest("该知识点的生成任务正在进行")
	}
	started := time.Now()
	status := &GenerationStatus{PointCode: point.Code, Status: "running", StartedAt: &started}
	g.jobs[point.Code] = status
	g.mu.Unlock()

	return g.generate(ctx, point)
}

func (g *ConceptGenerator) run(point *model.KnowledgePoint) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	_, _ = g.generate(ctx, point)
}

// generate 调 engine、落库，并更新任务状态。
func (g *ConceptGenerator) generate(ctx context.Context, point *model.KnowledgePoint) (*GenerationStatus, error) {
	concepts, err := g.runner.Generate(ctx, point)

	g.mu.Lock()
	status := g.jobs[point.Code]
	finished := time.Now()
	status.FinishedAt = &finished
	g.mu.Unlock()

	if err != nil {
		g.mu.Lock()
		status.Status = "failed"
		status.Error = err.Error()
		snapshot := *status
		g.mu.Unlock()
		return &snapshot, err
	}

	generated := 0
	for _, concept := range concepts {
		name := strings.TrimSpace(concept.Name)
		if name == "" {
			continue
		}
		if err := g.upsertConcept(point, name, concept.PageRank); err != nil {
			g.mu.Lock()
			status.Status = "failed"
			status.Error = err.Error()
			snapshot := *status
			g.mu.Unlock()
			return &snapshot, err
		}
		generated++
	}

	total, err := g.totalConcepts(point.Code)
	if err != nil {
		return nil, err
	}

	g.mu.Lock()
	status.Status = "done"
	status.Generated = generated
	status.Total = total
	snapshot := *status
	g.mu.Unlock()
	return &snapshot, nil
}

// upsertConcept 按「学科 + 概念名」去重：已有概念只更新 pageRank（保留原来源），新概念标记 generated。
func (g *ConceptGenerator) upsertConcept(point *model.KnowledgePoint, name string, pageRank float64) error {
	existing, err := g.knowledge.FindConceptByNamePoint(name, point.Code)
	if err != nil {
		return err
	}
	if existing != nil {
		return g.knowledge.UpsertConcept(&model.KnowledgeConcept{
			Code:        existing.Code,
			Name:        existing.Name,
			Field:       existing.Field,
			PointCode:   existing.PointCode,
			Explanation: existing.Explanation,
			Source:      existing.Source,
			PageRank:    float32(pageRank),
		})
	}

	return g.knowledge.UpsertConcept(&model.KnowledgeConcept{
		Code:        GeneratedConceptCode(point.Code, name),
		Name:        name,
		Field:       point.Name,
		PointCode:   point.Code,
		Level:       1,
		Explanation: "由 engine 的 generate.py 根据学科标题/描述生成",
		Source:      "generated",
		PageRank:    float32(pageRank),
	})
}

func (g *ConceptGenerator) totalConcepts(pointCode string) (int, error) {
	counts, err := g.knowledge.CountConceptsByPoint()
	if err != nil {
		return 0, err
	}
	return int(counts[pointCode]), nil
}

// GeneratedConceptCode 生成概念的稳定 code：同一学科同一概念名重复生成不会产生新行。
func GeneratedConceptCode(pointCode, name string) string {
	sum := sha1.Sum([]byte(pointCode + "|" + name))
	return fmt.Sprintf("GEN_%s_%s", hex.EncodeToString(sum[:])[:12], pointCode)
}
