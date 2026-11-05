package handler

import (
	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// DisciplineHandler 学科导航：主页面学科列表、学科详情与无资源时自动爬取
type DisciplineHandler struct {
	disciplines *service.DisciplineService
	crawler     *service.DisciplineCrawlService
}

func NewDisciplineHandler(
	disciplines *service.DisciplineService,
	crawler *service.DisciplineCrawlService,
) *DisciplineHandler {
	return &DisciplineHandler{disciplines: disciplines, crawler: crawler}
}

// List GET /disciplines
func (h *DisciplineHandler) List(c *gin.Context) {
	items, err := h.disciplines.List()
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, items)
}

// Detail GET /disciplines/:code?limit=12
func (h *DisciplineHandler) Detail(c *gin.Context) {
	limit, ok := queryPositiveInt(c, "limit", 12)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	detail, err := h.disciplines.Detail(c.Param("code"), limit)
	if err != nil {
		handleError(c, err)
		return
	}

	// 学科详情里的资源必须使用与知识点详情一致的列表 DTO。
	// 直接返回 GORM 模型会得到大写字段（ID/CreatedAt），前端按 resource.id
	// 取不到主键，RouterLink 会因缺少 id 参数而中断整棵路由组件的渲染。
	resources := make([]resourceListItem, 0, len(detail.Resources))
	for i := range detail.Resources {
		resources = append(resources, toResourceListItem(&detail.Resources[i]))
	}

	response.OK(c, gin.H{
		"discipline":     detail.Discipline,
		"points":         detail.Points,
		"resources":      resources,
		"resource_total": detail.ResourceTotal,
	})
}

// Crawl POST /disciplines/:code/crawl：无资源时自动去 B 站抓取
func (h *DisciplineHandler) Crawl(c *gin.Context) {
	status, err := h.crawler.Ensure(c.Param("code"))
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, status)
}

// CrawlStatus GET /disciplines/:code/crawl
func (h *DisciplineHandler) CrawlStatus(c *gin.Context) {
	status, err := h.crawler.Status(c.Param("code"))
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, status)
}
