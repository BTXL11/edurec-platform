package handler

import (
	"strconv"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// KnowledgeHandler 知识点导航：列表、图谱详情与知识点下的资源
type KnowledgeHandler struct {
	knowledge *service.KnowledgeService
}

func NewKnowledgeHandler(knowledge *service.KnowledgeService) *KnowledgeHandler {
	return &KnowledgeHandler{knowledge: knowledge}
}

// List GET /knowledge-points
func (h *KnowledgeHandler) List(c *gin.Context) {
	nodes, err := h.knowledge.List()
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, nodes)
}

// Detail GET /knowledge-points/:code?limit=12
func (h *KnowledgeHandler) Detail(c *gin.Context) {
	limit := 12
	if raw := c.Query("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 || value > 100 {
			response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
			return
		}
		limit = value
	}

	detail, err := h.knowledge.Detail(c.Param("code"), limit)
	if err != nil {
		handleError(c, err)
		return
	}

	items := make([]resourceListItem, 0, len(detail.Resources))
	for i := range detail.Resources {
		items = append(items, toResourceListItem(&detail.Resources[i]))
	}

	response.OK(c, gin.H{
		"point":          detail.Point,
		"nodes":          detail.Nodes,
		"edges":          detail.Edges,
		"resources":      items,
		"resource_total": detail.ResourceTotal,
		"concepts":       detail.Concepts,
		"concept_edges":  detail.ConceptEdges,
	})
}

// Resources GET /knowledge-points/:code/resources?page=1&page_size=12
func (h *KnowledgeHandler) Resources(c *gin.Context) {
	page, ok := queryPositiveInt(c, "page", 1)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	pageSize, ok := queryPositiveInt(c, "page_size", 12)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	items, total, node, err := h.knowledge.Resources(c.Param("code"), page, pageSize)
	if err != nil {
		handleError(c, err)
		return
	}

	list := make([]resourceListItem, 0, len(items))
	for i := range items {
		list = append(list, toResourceListItem(&items[i]))
	}

	response.OK(c, gin.H{
		"knowledge_point": node,
		"list":            list,
		"total":           total,
		"page":            page,
		"page_size":       pageSize,
		"has_more":        (page-1)*pageSize+len(list) < int(total),
	})
}
