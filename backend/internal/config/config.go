package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Shionyori/edurec-platform/backend/internal/model"
	"github.com/spf13/viper"
)

// Config 是应用的顶层配置
type Config struct {
	Server   ServerConfig             `mapstructure:"server"`
	Database DatabaseConfig           `mapstructure:"database"`
	Redis    RedisConfig              `mapstructure:"redis"`
	JWT      JWTConfig                `mapstructure:"jwt"`
	Engine   EngineConfig             `mapstructure:"engine"`
	Bilibili BilibiliConfig           `mapstructure:"bilibili"`
	Datasets map[string]DatasetConfig `mapstructure:"datasets"`
}

// 数据集字段映射支持的内部字段名。出现词表外的 key 一律报错：
// 拼错的 key 会静默产出空标题，从而让整批数据被跳过而看不出原因。
const (
	DatasetFieldTitle       = "title"
	DatasetFieldDescription = "description"
	DatasetFieldCoverURL    = "cover_url"
	DatasetFieldAuthor      = "author"
	DatasetFieldSourceURL   = "source_url"
	DatasetFieldCategory    = "category"
	DatasetFieldTags        = "tags"
	DatasetFieldViewCount   = "view_count"
)

// DatasetFormat 数据集文件的容器格式
const (
	DatasetFormatAuto  = "auto"
	DatasetFormatJSON  = "json"
	DatasetFormatJSONL = "jsonl"
	DatasetFormatCSV   = "csv"
)

// datasetFieldNames 映射词表，供校验与文档共用
var datasetFieldNames = []string{
	DatasetFieldTitle, DatasetFieldDescription, DatasetFieldCoverURL, DatasetFieldAuthor,
	DatasetFieldSourceURL, DatasetFieldCategory, DatasetFieldTags, DatasetFieldViewCount,
}

// DatasetConfig 第三方数据集导入配置：声明式地把外部字段映射到平台的采集结果契约。
// 数据集格式各家不同且会换，故字段对应关系全部外置到配置，换数据集只改 YAML 不改代码。
type DatasetConfig struct {
	File              string              `mapstructure:"file"`                // 数据集文件路径（相对 server 运行目录 backend/）
	Format            string              `mapstructure:"format"`              // auto | json | jsonl | csv，默认 auto 按扩展名与内容判定
	ItemsPath         string              `mapstructure:"items_path"`          // JSON 内记录数组所在路径（点号，如 data.list），空表示根即数组
	ResourceType      string              `mapstructure:"resource_type"`       // 落库的 resources.type：course | article | video
	DefaultCategory   string              `mapstructure:"default_category"`    // 分类字段为空时的兜底分类名
	SourceURLTemplate string              `mapstructure:"source_url_template"` // source_url 缺失时的兜底模板，{外部字段名} 作占位，如 https://x/learn/{obj_id}
	TagSeparator      string              `mapstructure:"tag_separator"`       // tags 为字符串时的分隔符，默认 ","
	Fields            map[string][]string `mapstructure:"fields"`              // 内部字段 → 外部字段候选，按序回退取第一个非空
	Metadata          []string            `mapstructure:"metadata"`            // 原样收进 metadata 的外部字段名
}

// Validate 校验单个数据集配置；配置错误必须在启动/导入前暴露，不能等到数据落库失败
func (d DatasetConfig) Validate(name string) error {
	if strings.TrimSpace(d.File) == "" {
		return fmt.Errorf("datasets.%s 缺少 file", name)
	}
	if !model.IsValidResourceType(d.ResourceType) {
		return fmt.Errorf("datasets.%s 的 resource_type %q 非法，只能是 course / article / video",
			name, d.ResourceType)
	}
	switch d.Format {
	case "", DatasetFormatAuto, DatasetFormatJSON, DatasetFormatJSONL, DatasetFormatCSV:
	default:
		return fmt.Errorf("datasets.%s 的 format %q 非法，只能是 auto / json / jsonl / csv", name, d.Format)
	}
	if len(d.Fields) == 0 {
		return fmt.Errorf("datasets.%s 缺少 fields 字段映射", name)
	}
	for field := range d.Fields {
		if !slices.Contains(datasetFieldNames, field) {
			return fmt.Errorf("datasets.%s 的 fields 含未知字段 %q，可用：%s",
				name, field, strings.Join(datasetFieldNames, " / "))
		}
	}
	return nil
}

// TagSeparatorOrDefault tags 为字符串时使用的分隔符
func (d DatasetConfig) TagSeparatorOrDefault() string {
	if d.TagSeparator == "" {
		return ","
	}
	return d.TagSeparator
}

// FormatOrDefault 归一化 format 取值
func (d DatasetConfig) FormatOrDefault() string {
	if d.Format == "" {
		return DatasetFormatAuto
	}
	return d.Format
}

// EngineConfig edurec-engine 接入配置
type EngineConfig struct {
	RecommendationsFile string `mapstructure:"recommendations_file"` // engine 输出的推荐结果 JSON 路径（路线 B 导入用）
	DatasetDir          string `mapstructure:"dataset_dir"`          // engine 演示/模拟数据集目录（demo_seed 播种用）
	SnapshotDir         string `mapstructure:"snapshot_dir"`         // 数据快照导出目录（export_snapshot 输出，engine 训练输入）
}

// BilibiliConfig 在线 B 站采集配置（搜索/评论实时爬取，见 backend/crawler/online.py）
type BilibiliConfig struct {
	PythonPath     string `mapstructure:"python_path"`      // Python 解释器，默认 "python"
	CrawlerDir     string `mapstructure:"crawler_dir"`      // backend/crawler 目录（online.py 所在，相对 server 运行目录 backend/）
	Category       string `mapstructure:"category"`         // 搜索落库分类名
	SearchLimit    int    `mapstructure:"search_limit"`     // 单次搜索导入条数
	SearchMaxPages int    `mapstructure:"search_max_pages"` // 无限滚动时最多翻 B 站页数，防无界爬取
	CommentLimit   int    `mapstructure:"comment_limit"`    // 单次评论抓取条数
	// AllowedTypenames 允许入库的 B 站分区名白名单。B 站搜索结果按关键词返回，
	// 非教育分区（影视剪辑/娱乐/游戏/音乐等）会随关键词一起被抓进来，必须在落库前拦掉。
	// 留空表示不限制（不推荐）。见 DefaultEducationalTypenames。
	AllowedTypenames []string `mapstructure:"allowed_typenames"`
}

// DefaultEducationalTypenames 教育向的 B 站分区白名单（配置未显式给出 allowed_typenames 时生效）。
//
// 取值依据（2026-09-19 实测两部分数据）：
//   - 清理后 edurec 库里保留资源的真实分区分布；
//   - 用「机器学习/雅思/高等数学/数据结构/数学/人工智能/数据分析/公开课/心理学/线性代数」
//     等教育关键词实际调用 B 站搜索，统计返回条目的分区。
//
// 收录原则：教学与知识类，以及 B 站对正经课程的高频误分类分区。
//   - 校园学习 / 计算机技术 / 科学科普 / 野生技能协会：明确的教学与知识分区；
//   - 人文历史 / 社科·法律·心理：通识学科内容；
//   - 日常 / 数码 / 运动文化 / 竞技体育：误分类重灾区（实测「数学分析」「泛函分析」
//     「高等代数」「数学建模」等课程被归入这些分区）；
//   - 软件应用 / 职业职场 / 科工机械 / 财经商业：技能与职业教育向
//     （实测「机器学习」「数据分析」「人工智能」的教程落在这里）。
//
// 刻意不收录：「其他」内容不可控；「预告·资讯」「原创音乐」等非教学分区。
// 未收录的分区一律不导入——新分区默认拦住，宁可漏也不要放回非教育内容。
var DefaultEducationalTypenames = []string{
	"校园学习",
	"计算机技术",
	"科学科普",
	"野生技能协会",
	"人文历史",
	"社科·法律·心理",
	"日常",
	"数码",
	"运动文化",
	"竞技体育",
	"软件应用",
	"职业职场",
	"科工机械",
	"财经商业",
}

// AllowedTypenamesOrDefault 返回生效的 B 站分区白名单：
// 配置里显式给了就用配置的，否则回退到 DefaultEducationalTypenames。
// 两者都为空表示不限制分区（不推荐）。
func (b BilibiliConfig) AllowedTypenamesOrDefault() []string {
	if len(b.AllowedTypenames) == 0 {
		return DefaultEducationalTypenames
	}
	return b.AllowedTypenames
}

// ServerConfig HTTP 服务器配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// DatabaseConfig MySQL 数据库配置
type DatabaseConfig struct {
	Driver       string `mapstructure:"driver"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	Charset      string `mapstructure:"charset"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

// DSN 返回 MySQL 连接字符串
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		d.Username, d.Password, d.Host, d.Port, d.Database, d.Charset)
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Addr 返回 Redis 地址
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// JWTConfig JWT 认证配置
type JWTConfig struct {
	AccessSecret  string `mapstructure:"access_secret"`
	RefreshSecret string `mapstructure:"refresh_secret"`
	AccessExpire  int    `mapstructure:"access_expire"`
	RefreshExpire int    `mapstructure:"refresh_expire"`
}

// Load 从 YAML 文件和环境变量加载配置
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 允许环境变量覆盖（将 . 替换为 _，前缀为空）
	// 例如 SERVER_PORT 会覆盖 server.port
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	if err := cfg.validateDatasets(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateDatasets 校验全部数据集配置。放在加载阶段是为了让拼错的字段映射
// 在启动时就报错，而不是等导入跑完才发现整批数据都因缺字段被跳过。
func (c *Config) validateDatasets() error {
	for name, dataset := range c.Datasets {
		if err := dataset.Validate(name); err != nil {
			return err
		}
	}
	return nil
}
