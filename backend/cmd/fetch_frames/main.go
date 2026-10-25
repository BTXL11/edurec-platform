package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/database"
	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/Shionyori/edurec-platform/backend/internal/repository"
	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

// fetch_frames：为已入库的 B 站资源补齐「封面 + 关键帧」。
//
// 数据来源是 B 站公开接口 /x/player/videoshot（进度条预览雪碧图），
// 不下载视频、不转存图片，只把雪碧图 URL 与网格参数写进 resources.metadata.frames。
//
// 用法（在 backend 目录下）：
//
//	CONFIG_PATH=configs/config.local.yaml go run ./cmd/fetch_frames [-limit 50] [-force] [-dry-run]
//
// 可反复执行：默认跳过已有 frames 的资源；-force 强制重抓。
func main() {
	limit := flag.Int("limit", 50, "本次最多补齐多少条资源，0 表示全部")
	force := flag.Bool("force", false, "已有 frames 的资源也重新抓取")
	dryRun := flag.Bool("dry-run", false, "只统计与调用，不写库")
	pythonFlag := flag.String("python", "", "Python 解释器（默认取配置 bilibili.python_path）")
	crawlerFlag := flag.String("crawler", "", "爬虫目录（默认取配置 bilibili.crawler_dir）")
	sleep := flag.Duration("sleep", 300*time.Millisecond, "每条之间的额外等待")
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
	repo := repository.NewResourceRepository(db)

	python := strings.TrimSpace(*pythonFlag)
	if python == "" {
		python = strings.TrimSpace(cfg.Bilibili.PythonPath)
	}
	if python == "" {
		python = "python"
	}
	crawlerDir := strings.TrimSpace(*crawlerFlag)
	if crawlerDir == "" {
		crawlerDir = strings.TrimSpace(cfg.Bilibili.CrawlerDir)
	}
	if crawlerDir == "" {
		crawlerDir = "crawler"
	}
	if _, err := os.Stat(crawlerDir); err != nil {
		log.Fatalf("爬虫目录不存在: %s（可用 -crawler 指定）", crawlerDir)
	}

	resources, err := loadAllResources(repo)
	if err != nil {
		log.Fatalf("读取资源失败: %v", err)
	}

	candidates := make([]model.Resource, 0, len(resources))
	for i := range resources {
		if service.BilibiliBvidFromURL(resources[i].SourceURL) == "" {
			continue
		}
		if !*force && hasFrames(resources[i].Metadata) {
			continue
		}
		candidates = append(candidates, resources[i])
	}

	total := len(candidates)
	if *limit > 0 && len(candidates) > *limit {
		candidates = candidates[:*limit]
	}
	fmt.Printf("[fetch_frames] 待补齐 %d 条（匹配 B 站资源共 %d 条，本次处理 %d 条）\n",
		total, len(resources), len(candidates))

	ok, skipped, failed := 0, 0, 0
	for index := range candidates {
		resource := &candidates[index]
		bvid := service.BilibiliBvidFromURL(resource.SourceURL)

		payload, err := fetchFrames(python, crawlerDir, bvid)
		if err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "[fetch_frames] %s 失败: %v\n", bvid, err)
			continue
		}
		if payload.Frames == nil || len(sprites(payload.Frames)) == 0 {
			skipped++
			continue
		}

		if !*dryRun {
			metadata, err := service.MergeFramesMetadata(resource.Metadata, payload.Frames)
			if err != nil {
				failed++
				fmt.Fprintf(os.Stderr, "[fetch_frames] %s 合并 metadata 失败: %v\n", bvid, err)
				continue
			}
			resource.Metadata = metadata
			if strings.TrimSpace(resource.CoverURL) == "" && payload.CoverURL != "" {
				resource.CoverURL = payload.CoverURL
			}
			if err := repo.Update(resource); err != nil {
				failed++
				fmt.Fprintf(os.Stderr, "[fetch_frames] %s 写库失败: %v\n", bvid, err)
				continue
			}
		}
		ok++
		if (index+1)%10 == 0 {
			fmt.Printf("[fetch_frames] 进度 %d/%d，成功 %d\n", index+1, len(candidates), ok)
		}
		if *sleep > 0 {
			time.Sleep(*sleep)
		}
	}

	prefix := "[fetch_frames]"
	if *dryRun {
		fmt.Printf("%s --dry-run：未写库\n", prefix)
	}
	fmt.Printf("%s 完成：成功 %d，无关键帧 %d，失败 %d\n", prefix, ok, skipped, failed)
}

type framePayload struct {
	Bvid     string         `json:"bvid"`
	CoverURL string         `json:"cover_url"`
	Frames   map[string]any `json:"frames"`
}

// fetchFrames 调 crawler/frames.py 取单条视频的封面与关键帧
func fetchFrames(python, crawlerDir, bvid string) (*framePayload, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, python, "frames.py", "--bvid", bvid)
	cmd.Dir = crawlerDir
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("%s", detail)
	}

	var payload framePayload
	if err := json.Unmarshal([]byte(stdout.String()), &payload); err != nil {
		return nil, fmt.Errorf("解析 frames.py 输出失败: %w", err)
	}
	return &payload, nil
}

func sprites(frames map[string]any) []any {
	raw, ok := frames["sprites"]
	if !ok {
		return nil
	}
	items, _ := raw.([]any)
	return items
}

func hasFrames(metadata string) bool {
	return strings.Contains(metadata, `"frames"`)
}

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
