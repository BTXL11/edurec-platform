package service

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
)

// CrawlImportService 把外部采集结果导入资源表，是各来源共用的落库内核。
//
// 调用方负责把来源数据整理成 CrawlItem（B 站走 crawler 输出的交接文件，
// 第三方数据集走 LoadDataset 的字段映射），本服务只管落库：
// 按 source_url 判重、分类 find-or-create、字段截断。因此同一个来源可反复导入。
//
// 资源以「普通 Resource」身份入库，与手工录入的资源混在同一列表里；已存在时
// 不新增行、只刷新动态字段（播放量、扩展信息），不覆盖平台侧字段（保证幂等、
// 不冲掉人工编辑）。
type CrawlImportService struct {
	resources  repository.ResourceRepository
	categories repository.CategoryRepository
	filePath   string
}

func NewCrawlImportService(
	resources repository.ResourceRepository,
	categories repository.CategoryRepository,
	filePath string,
) *CrawlImportService {
	return &CrawlImportService{resources: resources, categories: categories, filePath: filePath}
}

// CrawlImportResult 导入统计
type CrawlImportResult struct {
	CreatedResources  int `json:"created_resources"`
	UpdatedResources  int `json:"updated_resources"`
	SkippedResources  int `json:"skipped_resources"` // 缺必填字段，未落库
	CreatedCategories int `json:"created_categories"`
}

// 各来源在资源表里的固定 type；model.Resource.Type 允许 course / article / video
const (
	bilibiliResourceType = "video"
)

// bilibiliSourceTemplate 与 crawler/collect.py 的 SOURCE_URL_TEMPLATE 保持一致。
// comment.go 依赖它从 source_url 反解 BV 号，不要改动其值。
const bilibiliSourceTemplate = "https://www.bilibili.com/video/"

// BilibiliImportOptions B 站来源的落库参数
var BilibiliImportOptions = CrawlImportOptions{
	ResourceType:      bilibiliResourceType,
	SourceURLTemplate: bilibiliSourceTemplate,
}

// model.Resource 各字段的 gorm size，超长按字符截断，避免整批导入因单条超长而失败
const (
	maxTitleRunes     = 256
	maxCoverURLRunes  = 512
	maxAuthorRunes    = 128
	maxSourceURLRunes = 512
)

// crawlFile 是 crawler/run.py 写出的交接文件结构
type crawlFile struct {
	Version int         `json:"version"`
	Source  string      `json:"source"`
	Items   []CrawlItem `json:"items"`
}

// CrawlItem 是各来源共用的采集结果单元。
//
// 身份键优先取 SourceID；B 站的采集脚本只发 bvid，故 bvid 作为兼容别名保留，
// 二者取到任意一个即可入库。sourceURL 为空时由来源的 URL 模板 + 身份键兜底。
type CrawlItem struct {
	SourceID    string         `json:"source_id"`
	Bvid        string         `json:"bvid"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	CoverURL    string         `json:"cover_url"`
	Author      string         `json:"author"`
	SourceURL   string         `json:"source_url"`
	Category    string         `json:"category"`
	Tags        []string       `json:"tags"`
	ViewCount   uint           `json:"view_count"`
	Metadata    map[string]any `json:"metadata"`
}

// sourceID 返回本条记录的身份键：优先 source_id，回退到 B 站兼容字段 bvid
func (item CrawlItem) sourceID() string {
	if id := strings.TrimSpace(item.SourceID); id != "" {
		return id
	}
	return strings.TrimSpace(item.Bvid)
}

// crawlRecord 是校验、截断、序列化之后的待落库记录
type crawlRecord struct {
	title        string
	description  string
	coverURL     string
	author       string
	sourceURL    string
	category     string
	tagsJSON     string
	metadataJSON string
	viewCount    uint
}

// Import 读取爬虫输出文件并落库
func (s *CrawlImportService) Import() (*CrawlImportResult, error) {
	items, err := s.readFileItems()
	if err != nil {
		return nil, err
	}
	result, _, err := s.ImportItems(items, BilibiliImportOptions, true)
	return result, err
}

// Preview 走与 Import 完全相同的解析、判重与分类逻辑，但不写库，用于 -dry-run
func (s *CrawlImportService) Preview() (*CrawlImportResult, error) {
	items, err := s.readFileItems()
	if err != nil {
		return nil, err
	}
	result, _, err := s.ImportItems(items, BilibiliImportOptions, false)
	return result, err
}

// readFileItems 读取爬虫输出文件并反序列化为 item 列表
func (s *CrawlImportService) readFileItems() ([]CrawlItem, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	var file crawlFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, apperror.Internal(err)
	}
	return file.Items, nil
}

// CrawlImportOptions 落库参数。各来源的差异只有这两项：落库的 type，
// 以及 item 未给出 source_url 时用什么 URL 模板 + 身份键兜底。
type CrawlImportOptions struct {
	ResourceType      string // 落库的 model.Resource.Type
	SourceURLTemplate string // 例 "https://www.bilibili.com/video/"；为空且 item 无 source_url 则跳过该条
}

// ImportItems 把已解析的 item 落库（write=false 时只统计不写库）。
// 供离线导入（Import/Preview）、在线搜索爬取（BilibiliOnlineService）与数据集导入
// （cmd/import_dataset）复用。
//
// 返回的 resources 按输入顺序给出已落库的资源行（created 经 Create 后已有 ID；updated 为判重命中行），
// 供在线搜索把新内容回传给前端追加展示；离线导入可忽略。
func (s *CrawlImportService) ImportItems(
	items []CrawlItem,
	opts CrawlImportOptions,
	write bool,
) (*CrawlImportResult, []model.Resource, error) {
	result := &CrawlImportResult{}
	records := make([]crawlRecord, 0, len(items))
	for _, item := range items {
		record, ok := newCrawlRecord(item, opts.SourceURLTemplate)
		if !ok {
			result.SkippedResources++
			continue
		}
		records = append(records, record)
	}
	if len(records) == 0 {
		return result, nil, nil
	}

	// 一次性查出已存在的来源链接，避免逐条查询
	urls := make([]string, 0, len(records))
	for i := range records {
		urls = append(urls, records[i].sourceURL)
	}
	existing, err := s.resources.FindBySourceURLs(urls)
	if err != nil {
		return nil, nil, apperror.Internal(err)
	}
	existingByURL := make(map[string]*model.Resource, len(existing))
	for i := range existing {
		existingByURL[existing[i].SourceURL] = &existing[i]
	}

	categoryIDs := map[string]uint{}
	imported := make([]model.Resource, 0, len(records))
	for i := range records {
		record := &records[i]

		categoryID, err := s.resolveCategory(record.category, categoryIDs, result, write)
		if err != nil {
			return nil, nil, err
		}

		if current, ok := existingByURL[record.sourceURL]; ok {
			// 已存在：不新增行，只刷新动态字段（播放量、扩展信息）
			current.ViewCount = record.viewCount
			current.Metadata = record.metadataJSON
			if write {
				if err := s.resources.Update(current); err != nil {
					return nil, nil, apperror.Internal(err)
				}
			}
			result.UpdatedResources++
			imported = append(imported, *current)
			continue
		}

		resource := record.toResource(categoryID, opts.ResourceType)
		if write {
			if err := s.resources.Create(resource); err != nil {
				return nil, nil, apperror.Internal(err)
			}
		}
		// 文件内若出现重复的 source_url，后续条目按「已存在」处理
		existingByURL[record.sourceURL] = resource
		result.CreatedResources++
		imported = append(imported, *resource)
	}

	return result, imported, nil
}

// resolveCategory 按名称取分类 ID，不存在则创建；同名分类在单次导入内只查一次
func (s *CrawlImportService) resolveCategory(
	name string,
	cache map[string]uint,
	result *CrawlImportResult,
	write bool,
) (uint, error) {
	if id, ok := cache[name]; ok {
		return id, nil
	}

	category, err := s.categories.FindByName(name)
	if err == nil {
		cache[name] = category.ID
		return category.ID, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return 0, apperror.Internal(err)
	}

	if !write {
		cache[name] = 0
		result.CreatedCategories++
		return 0, nil
	}

	created := &model.Category{Name: name}
	if err := s.categories.Create(created); err != nil {
		return 0, apperror.Internal(err)
	}
	cache[name] = created.ID
	result.CreatedCategories++
	return created.ID, nil
}

// newCrawlRecord 校验必填字段并归一化；缺字段返回 ok=false，由调用方计入跳过。
//
// source_url 是判重的唯一依据，因此它必须能确定下来：item 自带则用自带的，
// 否则由来源的 URL 模板 + 身份键拼出。两者都给不出（模板缺失或身份键为空）时
// 必须跳过——否则每次导入都会重复插入同一行。
func newCrawlRecord(item CrawlItem, sourceURLTemplate string) (crawlRecord, bool) {
	title := strings.TrimSpace(item.Title)
	category := strings.TrimSpace(item.Category)
	if title == "" || category == "" {
		return crawlRecord{}, false
	}

	sourceURL := strings.TrimSpace(item.SourceURL)
	if sourceURL == "" {
		sourceID := item.sourceID()
		if sourceID == "" || sourceURLTemplate == "" {
			return crawlRecord{}, false
		}
		sourceURL = sourceURLTemplate + sourceID
	}

	// Tags 列是 JSON 数组，nil 会被 Marshal 成 null，这里兜底成 []
	if item.Tags == nil {
		item.Tags = []string{}
	}
	tagsJSON, err := json.Marshal(item.Tags)
	if err != nil {
		return crawlRecord{}, false
	}

	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(item.Metadata)
	if err != nil {
		return crawlRecord{}, false
	}

	return crawlRecord{
		title:        truncateRunes(title, maxTitleRunes),
		description:  strings.TrimSpace(item.Description),
		coverURL:     truncateRunes(strings.TrimSpace(item.CoverURL), maxCoverURLRunes),
		author:       truncateRunes(strings.TrimSpace(item.Author), maxAuthorRunes),
		sourceURL:    truncateRunes(sourceURL, maxSourceURLRunes),
		category:     category,
		tagsJSON:     string(tagsJSON),
		metadataJSON: string(metadataJSON),
		viewCount:    item.ViewCount,
	}, true
}

func (r crawlRecord) toResource(categoryID uint, resourceType string) *model.Resource {
	return &model.Resource{
		Title:       r.title,
		Description: r.description,
		CoverURL:    r.coverURL,
		Type:        resourceType,
		CategoryID:  categoryID,
		Tags:        r.tagsJSON,
		Metadata:    r.metadataJSON,
		Author:      r.author,
		SourceURL:   r.sourceURL,
		ViewCount:   r.viewCount,
	}
}

// truncateRunes 按字符截断（而非字节），与 MySQL varchar 的长度语义一致
func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
