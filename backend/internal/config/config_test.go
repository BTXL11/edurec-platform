package config_test

import (
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
)

// 默认教育分区白名单必须覆盖实测确认属于教育内容的 B 站分区。
//
// 背景：B 站搜索结果会按关键词混入非教育内容（影视/娱乐/游戏/音乐等），
// 因此落库前用 DefaultEducationalTypenames 做白名单拦截。但 B 站对内容的
// 分区归类很不规整——大量正经课程被归进「日常」「运动文化」「竞技体育」等
// 非教学分区，技能类教程又落在「软件应用」「职业职场」。这个用例把实测确认
// 属于教育内容的分区钉住，防止后续有人收紧白名单时把正常课程一起拦掉。
func TestDefaultEducationalTypenamesCoversVerifiedEducationalSections(t *testing.T) {
	// 来自 2026-09-19 两处实测：清理后 edurec 库的分区分布，
	// 以及用教育关键词调用 B 站搜索返回的分区统计
	verifiedEducational := []string{
		"校园学习",     // 课程主力分区
		"计算机技术",    // 编程/算法课程
		"科学科普",     // 数学/物理解说
		"野生技能协会",   // 技能教程（实测含机器学习、ROS、数据分析）
		"人文历史",     // 通识（实测含解析几何史、雅思流程、心理学）
		"社科·法律·心理", // 通识（实测含概率论、心理学）
		"日常",       // 误分类重灾区：数学分析、泛函分析、高等代数均在此
		"数码",       // 实测含高等数学习题讲解
		"运动文化",     // 实测含数学建模
		"竞技体育",     // 实测含数学建模国赛
		"软件应用",     // 实测含机器学习/数据分析软件教程
		"职业职场",     // 实测含人工智能、数据分析课程
		"科工机械",     // 实测含人工智能硬件相关内容
		"财经商业",     // 实测含人工智能商业分析内容
	}

	allowed := make(map[string]bool, len(config.DefaultEducationalTypenames))
	for _, name := range config.DefaultEducationalTypenames {
		allowed[name] = true
	}

	for _, name := range verifiedEducational {
		if !allowed[name] {
			t.Errorf("默认白名单缺少教育分区 %q：该分区下有真实课程，收紧后会误杀", name)
		}
	}
}

// 反向确认：典型非教育分区，以及内容不可控的分区，不应出现在默认白名单里
func TestDefaultEducationalTypenamesExcludesNonEducationalSections(t *testing.T) {
	nonEducational := []string{
		"影视剪辑", "影视杂谈", "娱乐粉丝创作", "娱乐杂谈",
		"明星综合", "网络游戏", "音乐综合", "原创音乐",
		"预告·资讯", "其他",
	}

	allowed := make(map[string]bool, len(config.DefaultEducationalTypenames))
	for _, name := range config.DefaultEducationalTypenames {
		allowed[name] = true
	}

	for _, name := range nonEducational {
		if allowed[name] {
			t.Errorf("默认白名单不应包含非教育/不可控分区 %q", name)
		}
	}
}

// 配置未显式给出白名单时应回退到默认值；显式给出时以配置为准
func TestAllowedTypenamesOrDefault(t *testing.T) {
	t.Run("未配置时回退默认", func(t *testing.T) {
		cfg := config.BilibiliConfig{}
		got := cfg.AllowedTypenamesOrDefault()
		if len(got) != len(config.DefaultEducationalTypenames) {
			t.Fatalf("len = %d, want %d", len(got), len(config.DefaultEducationalTypenames))
		}
	})

	t.Run("显式配置时以配置为准", func(t *testing.T) {
		cfg := config.BilibiliConfig{AllowedTypenames: []string{"校园学习"}}
		got := cfg.AllowedTypenamesOrDefault()
		if len(got) != 1 || got[0] != "校园学习" {
			t.Fatalf("got = %v, want [校园学习]", got)
		}
	})
}
