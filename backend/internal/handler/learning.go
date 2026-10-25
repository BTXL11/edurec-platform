package handler

import (
	"strconv"

	"github.com/Shionyori/edurec-platform/backend/internal/apperror"
	"github.com/Shionyori/edurec-platform/backend/internal/middleware"
	"github.com/Shionyori/edurec-platform/backend/internal/response"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
	"github.com/gin-gonic/gin"
)

// LearningHandler 自适应学习：问卷、知识画像、学习路径与逐节解锁
type LearningHandler struct {
	learning *service.LearningService
}

func NewLearningHandler(learning *service.LearningService) *LearningHandler {
	return &LearningHandler{learning: learning}
}

// Profile GET /learning-profile
func (h *LearningHandler) Profile(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证")
		return
	}
	items, err := h.learning.Profile(userID)
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, items)
}

type questionnaireRequest struct {
	// 知识点 code -> 自评等级 0-3（0 完全没接触，3 很熟练）
	Answers map[string]int `json:"answers" binding:"required"`
}

// SubmitQuestionnaire POST /learning-profile/questionnaire
func (h *LearningHandler) SubmitQuestionnaire(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证")
		return
	}
	var req questionnaireRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	items, err := h.learning.SubmitQuestionnaire(userID, req.Answers)
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, items)
}

type createPathRequest struct {
	Target string `json:"target" binding:"required"`
}

// CreatePath POST /learning-paths
func (h *LearningHandler) CreatePath(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证")
		return
	}
	var req createPathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	path, err := h.learning.BuildPath(userID, req.Target)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Created(c, path)
}

// GetPath GET /learning-paths/:id
func (h *LearningHandler) GetPath(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证")
		return
	}
	pathID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || pathID == 0 {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	path, err := h.learning.GetPath(userID, uint(pathID))
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, path)
}

type completeStepRequest struct {
	Index int `json:"index"`
}

// CompleteStep POST /learning-paths/:id/complete
func (h *LearningHandler) CompleteStep(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证")
		return
	}
	pathID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || pathID == 0 {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	var req completeStepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	path, err := h.learning.CompleteStep(userID, uint(pathID), req.Index)
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, path)
}

// Resources GET /learning-paths/:id/resources?limit=12
func (h *LearningHandler) Resources(c *gin.Context) {
	userID, ok := middleware.UserID(c)
	if !ok {
		response.Error(c, 401, apperror.CodeUnauthorized, "未认证")
		return
	}
	pathID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || pathID == 0 {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}
	limit, ok := queryPositiveInt(c, "limit", 12)
	if !ok {
		response.Error(c, 400, apperror.CodeBadRequest, "请求参数错误")
		return
	}

	point, items, err := h.learning.PathResources(userID, uint(pathID), limit)
	if err != nil {
		handleError(c, err)
		return
	}
	list := make([]resourceListItem, 0, len(items))
	for i := range items {
		list = append(list, toResourceListItem(&items[i]))
	}
	response.OK(c, gin.H{
		"knowledge_point": point,
		"list":            list,
		"total":           len(list),
	})
}
