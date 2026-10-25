package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/database"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

// seed_knowledge：建立两层知识图谱。
//
//	学科层：16 个学科节点 + 先修关系 + 按关键词关联平台资源（本仓库自有数据）
//	概念层：从 EMBEDD-ER / MOOCCube 导入概念与先修边，并按「课程 -> 概念」映射归到学科
//
// 用法（在 backend 目录下）：
//
//	CONFIG_PATH=configs/config.local.yaml go run ./cmd/seed_knowledge [-dry-run] [-source <MOOCCube 目录>]
//
// 数据源路径优先取 -source，其次环境变量 KNOWLEDGE_SOURCE_DIR，最后配置 knowledge.source_dir；
// 项目迁入 engine 后只需切换这一处路径（scripts/set-kg-source.ps1 一键修改）。
//
// 可反复执行：知识点/概念按 code、先修关系按 (from,to)、资源关联按 (resource,point) 幂等写入。
func main() {
	dryRun := flag.Bool("dry-run", false, "只统计匹配结果，不写库")
	minScore := flag.Float64("min-score", 0.5, "资源与知识点的最低匹配分（0-1）")
	source := flag.String("source", "", "MOOCCube 数据目录（覆盖配置 knowledge.source_dir）")
	skipConcepts := flag.Bool("skip-concepts", false, "只建学科层，不导入概念图")
	reset := flag.Bool("reset", false, "重建前清空知识图谱派生数据（学科/概念/先修/资源关联），资源与用户数据不动")
	flag.Parse()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	db, err := database.InitMySQL(cfg.Database)
	if err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}

	knowledgeRepo := repository.NewKnowledgeRepository(db)
	resourceRepo := repository.NewResourceRepository(db)

	// 重复构建：先清掉上一版派生数据，避免旧关联（已不匹配的资源/概念）残留。
	// 资源、用户、行为、评分不受影响。
	if *reset && !*dryRun {
		if err := knowledgeRepo.ResetKnowledgeGraph(); err != nil {
			log.Fatalf("清空旧图谱失败: %v", err)
		}
		fmt.Println("[seed_knowledge] 已清空旧图谱数据，开始重建")
	}

	resources, err := loadAllResources(resourceRepo)
	if err != nil {
		log.Fatalf("读取资源失败: %v", err)
	}

	// 先验证图本身合法（有环会直接失败），避免把坏图写进库
	codes := make([]string, 0, len(service.DefaultKnowledgePoints))
	for _, seed := range service.DefaultKnowledgePoints {
		codes = append(codes, seed.Code)
	}
	levels, err := service.ComputeLevels(codes, service.DefaultKnowledgeEdges)
	if err != nil {
		log.Fatalf("先修关系不合法: %v", err)
	}

	codeToID := make(map[string]uint, len(codes))
	totalLinks := 0
	for _, seed := range service.DefaultKnowledgePoints {
		point := &model.KnowledgePoint{
			Code:        seed.Code,
			Name:        seed.Name,
			Description: seed.Description,
			Level:       levels[seed.Code],
			SortOrder:   len(codeToID),
		}
		if !*dryRun {
			if err := knowledgeRepo.UpsertPoint(point); err != nil {
				log.Fatalf("写入知识点 %s 失败: %v", seed.Code, err)
			}
			saved, err := knowledgeRepo.FindPointByCode(seed.Code)
			if err != nil || saved == nil {
				log.Fatalf("回查知识点 %s 失败: %v", seed.Code, err)
			}
			codeToID[seed.Code] = saved.ID
		}

		linked := 0
		for i := range resources {
			score := service.MatchScore(
				resources[i].Title,
				parseTags(resources[i].Tags),
				resources[i].Description,
				seed.Keywords,
			)
			if score < *minScore {
				continue
			}
			linked++
			if *dryRun {
				continue
			}
			if err := knowledgeRepo.UpsertResourceLink(&model.ResourceKnowledgePoint{
				ResourceID: resources[i].ID,
				PointID:    codeToID[seed.Code],
				Weight:     float32(score),
			}); err != nil {
				log.Fatalf("关联资源 %d 到知识点 %s 失败: %v", resources[i].ID, seed.Code, err)
			}
		}
		totalLinks += linked
		fmt.Printf("[seed_knowledge] %-22s level=%d 资源=%d\n", seed.Code, levels[seed.Code], linked)
	}

	edgeCount := 0
	for _, edge := range service.DefaultKnowledgeEdges {
		from, okFrom := codeToID[edge.From]
		to, okTo := codeToID[edge.To]
		if *dryRun {
			edgeCount++
			continue
		}
		if !okFrom || !okTo {
			log.Fatalf("先修关系引用了不存在的知识点: %s -> %s", edge.From, edge.To)
		}
		if err := knowledgeRepo.CreateEdgeIfAbsent(&model.KnowledgePointEdge{FromID: from, ToID: to}); err != nil {
			log.Fatalf("写入先修关系失败: %v", err)
		}
		edgeCount++
	}

	for code, level := range levels {
		if *dryRun {
			continue
		}
		if err := knowledgeRepo.UpdatePointLevel(codeToID[code], level); err != nil {
			log.Fatalf("更新知识点层级失败: %v", err)
		}
	}

	prefix := "[seed_knowledge]"
	if *dryRun {
		fmt.Printf("%s --dry-run：未写库\n", prefix)
	}
	fmt.Printf("%s 学科层：%d 个学科，%d 条先修，%d 条资源关联（扫描资源 %d 个）\n",
		prefix, len(service.DefaultKnowledgePoints), edgeCount, totalLinks, len(resources))

	if *skipConcepts {
		return
	}

	sourceDir := strings.TrimSpace(*source)
	if sourceDir == "" {
		sourceDir = strings.TrimSpace(cfg.Knowledge.SourceDir)
	}
	if sourceDir == "" {
		fmt.Printf("%s 未配置 knowledge.source_dir，跳过概念层导入\n", prefix)
		return
	}
	if err := importConcepts(sourceDir, knowledgeRepo, *dryRun); err != nil {
		log.Fatalf("导入概念图失败: %v", err)
	}
}

// importConcepts 从 MOOCCube 关系文件导入概念层，并按课程名把概念归到学科。
func importConcepts(sourceDir string, repo *repository.KnowledgeRepo, dryRun bool) error {
	prefix := "[seed_knowledge][concepts]"

	prereqPath := filepath.Join(sourceDir, "relations", "prerequisite-dependency.json")
	conceptPath := filepath.Join(sourceDir, "entities", "concept.json")
	fieldPath := filepath.Join(sourceDir, "relations", "concept-field.json")
	coursePath := filepath.Join(sourceDir, "entities", "course.json")
	courseConceptPath := filepath.Join(sourceDir, "relations", "course-concept.json")

	for _, path := range []string{prereqPath, conceptPath, fieldPath, coursePath, courseConceptPath} {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("缺少数据文件 %s（可用 scripts/set-kg-source.ps1 切换路径）: %w", path, err)
		}
	}

	edges, err := readTSVPairs(prereqPath)
	if err != nil {
		return err
	}
	if len(edges) == 0 {
		return fmt.Errorf("先修关系为空: %s", prereqPath)
	}

	// 节点集合 = 先修边两端
	nodeSet := make(map[string]struct{}, len(edges)*2)
	for _, edge := range edges {
		nodeSet[edge[0]] = struct{}{}
		nodeSet[edge[1]] = struct{}{}
	}
	conceptIDs := make([]string, 0, len(nodeSet))
	for id := range nodeSet {
		conceptIDs = append(conceptIDs, id)
	}

	names, explanations, err := readConceptEntities(conceptPath, nodeSet)
	if err != nil {
		return err
	}
	fields, err := readConceptFields(fieldPath, nodeSet)
	if err != nil {
		return err
	}
	courseNames, err := readCourseNames(coursePath)
	if err != nil {
		return err
	}
	courseConcepts, err := readCourseConcepts(courseConceptPath)
	if err != nil {
		return err
	}

	assigned := service.AssignConceptSubjects(conceptIDs, courseConcepts, courseNames)

	// 概念层级：MOOCCube 原始数据含环，用 SCC 缩点的容错算法（同环内概念同级）
	edgeSeeds := make([]service.KnowledgeEdgeSeed, 0, len(edges))
	for _, edge := range edges {
		edgeSeeds = append(edgeSeeds, service.KnowledgeEdgeSeed{From: edge[0], To: edge[1]})
	}
	levels := service.ComputeLevelsTolerant(conceptIDs, edgeSeeds)

	codeToID := make(map[string]uint, len(conceptIDs))
	assignedCount := 0
	for _, code := range conceptIDs {
		pointCode := assigned[code]
		if pointCode != "" {
			assignedCount++
		}
		concept := &model.KnowledgeConcept{
			Code:        code,
			Name:        names[code],
			Field:       fields[code],
			PointCode:   pointCode,
			Level:       levels[code],
			Explanation: explanations[code],
		}
		if dryRun {
			continue
		}
		if err := repo.UpsertConcept(concept); err != nil {
			return fmt.Errorf("写入概念 %s 失败: %w", code, err)
		}
		saved, err := repo.FindConceptByCode(code)
		if err != nil || saved == nil {
			return fmt.Errorf("回查概念 %s 失败: %v", code, err)
		}
		codeToID[code] = saved.ID
	}

	if !dryRun {
		for _, edge := range edges {
			from, okFrom := codeToID[edge[0]]
			to, okTo := codeToID[edge[1]]
			if !okFrom || !okTo {
				continue
			}
			if err := repo.UpsertConceptEdge(&model.KnowledgeConceptEdge{FromID: from, ToID: to}); err != nil {
				return fmt.Errorf("写入概念先修关系失败: %w", err)
			}
		}
		for code, level := range levels {
			if err := repo.UpdateConceptLevel(codeToID[code], level); err != nil {
				return fmt.Errorf("更新概念层级失败: %w", err)
			}
		}
	}

	fmt.Printf("%s 概念 %d 个，先修 %d 条，归入学科 %d 个（源：%s）\n",
		prefix, len(conceptIDs), len(edges), assignedCount, sourceDir)
	return nil
}

// readTSVPairs 读取两列 TSV（.json 后缀但实际是 TSV，见 EMBEDD-ER recommender/stats.py）
func readTSVPairs(path string) ([][2]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开 %s 失败: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 4*1024*1024)
	out := make([][2]string, 0, 2048)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		left := strings.TrimSpace(parts[0])
		right := strings.TrimSpace(parts[1])
		if left == "" || right == "" {
			continue
		}
		out = append(out, [2]string{left, right})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 %s 失败: %w", path, err)
	}
	return out, nil
}

// readConceptEntities 流式读取概念实体（JSON Lines），只保留节点集合内的概念
func readConceptEntities(path string, wanted map[string]struct{}) (map[string]string, map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("打开 %s 失败: %w", path, err)
	}
	defer file.Close()

	names := make(map[string]string, len(wanted))
	explanations := make(map[string]string, len(wanted))

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entity struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Explanation string `json:"explanation"`
		}
		if err := json.Unmarshal([]byte(line), &entity); err != nil {
			continue
		}
		if _, ok := wanted[entity.ID]; !ok {
			continue
		}
		names[entity.ID] = entity.Name
		explanations[entity.ID] = entity.Explanation
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("读取 %s 失败: %w", path, err)
	}
	return names, explanations, nil
}

// readConceptFields 概念 -> 领域。concept-field 的值形如 K_T_数学_数学，取第 3 段作领域名。
func readConceptFields(path string, wanted map[string]struct{}) (map[string]string, error) {
	pairs, err := readTSVPairs(path)
	if err != nil {
		return nil, err
	}
	fields := make(map[string]string, len(wanted))
	for _, pair := range pairs {
		concept, fieldID := pair[0], pair[1]
		if _, ok := wanted[concept]; !ok {
			continue
		}
		parts := strings.Split(fieldID, "_")
		if len(parts) >= 4 {
			fields[concept] = parts[2]
		}
	}
	return fields, nil
}

// readCourseNames 读取课程 ID -> 课程名
func readCourseNames(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开 %s 失败: %w", path, err)
	}
	defer file.Close()

	names := make(map[string]string, 1024)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var course struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(line), &course); err != nil {
			continue
		}
		if course.ID != "" {
			names[course.ID] = course.Name
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取 %s 失败: %w", path, err)
	}
	return names, nil
}

// readCourseConcepts 读取课程 -> 概念集合
func readCourseConcepts(path string) (map[string][]string, error) {
	pairs, err := readTSVPairs(path)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]string, 1024)
	for _, pair := range pairs {
		course, concept := pair[0], pair[1]
		out[course] = append(out[course], concept)
	}
	return out, nil
}

// loadAllResources 分页读取全部资源（知识点关联需要全量扫描标题与标签）
func loadAllResources(repo *repository.ResourceRepo) ([]model.Resource, error) {
	const pageSize = 100
	var all []model.Resource
	for page := 1; ; page++ {
		result, err := repo.List(repository.ResourceListQuery{Page: page, PageSize: pageSize, Sort: "latest"})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Items...)
		if len(result.Items) == 0 || int64(len(all)) >= result.Total {
			break
		}
	}
	return all, nil
}

func parseTags(raw string) []string {
	if raw == "" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil
	}
	return tags
}
