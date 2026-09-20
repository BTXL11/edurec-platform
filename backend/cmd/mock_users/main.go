package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/Shionyori/edurec-platform/backend/internal/config"
	"github.com/Shionyori/edurec-platform/backend/internal/database"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// mock_users：生成模拟用户，让他们对【现有真实资源】产生行为。
//
// 与 demo_seed 的区别：demo_seed 会引入 sim 的 500 个模拟资源；
// 本命令复用库里现有的资源（如 B 站采集的视频），只造用户 + 行为 + 评分，
// 用于"真实资源 + 模拟用户"的混合场景（跳过冷启动，又不污染资源表）。
//
// 每个资源至少被 -min-item 个用户交互（默认 5，对齐 engine 的 min_item_interactions），
// 保证 engine 训练时不会被 clean 过滤掉。
//
// 用法：CONFIG_PATH=configs/config.yaml go run ./cmd/mock_users -users 50
func main() {
	users := flag.Int("users", 50, "生成模拟用户数量")
	password := flag.String("password", "demo123456", "模拟用户统一密码")
	minItem := flag.Int("min-item", 5, "每个资源至少被多少用户交互（对齐 engine min_item_interactions）")
	seed := flag.Int64("seed", 42, "随机种子（可复现）")
	admin := flag.Bool("admin", true, "确保存在管理员 demo_admin")
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

	// 现有真实资源
	var resIDs []uint
	if err := db.Raw("SELECT id FROM resources WHERE deleted_at IS NULL ORDER BY id").Scan(&resIDs).Error; err != nil {
		log.Fatalf("查询资源失败: %v", err)
	}
	if len(resIDs) == 0 {
		log.Fatal("库里没有资源，请先采集（scripts/bilibili.sh）或导入资源")
	}

	rng := rand.New(rand.NewSource(*seed))
	now := time.Now()

	// 生成用户
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("生成密码哈希失败: %v", err)
	}
	usrOut := make([][]any, 0, *users)
	for i := 0; i < *users; i++ {
		usrOut = append(usrOut, []any{
			fmt.Sprintf("mock%d", i+1),
			fmt.Sprintf("mock%d@mock.local", i+1),
			string(hash),
			fmt.Sprintf("模拟用户%d", i+1),
			now,
		})
	}
	insertIgnore(db, "users",
		[]string{"username", "email", "password_hash", "display_name", "created_at"},
		usrOut, 300)

	// 查生成的用户 ID
	var userIDs []uint
	if err := db.Raw("SELECT id FROM users WHERE username LIKE 'mock%' ORDER BY id").Scan(&userIDs).Error; err != nil {
		log.Fatalf("查询用户失败: %v", err)
	}
	if len(userIDs) < *minItem {
		log.Fatalf("用户数 %d 少于 min-item=%d，请增大 -users", len(userIDs), *minItem)
	}

	// 生成行为：每个资源 × minItem 个随机用户
	actions := []string{"view", "click", "favorite"}
	behOut := make([][]any, 0, len(resIDs)*(*minItem))
	ratOut := make([][]any, 0)
	for _, rid := range resIDs {
		for _, uid := range pickN(rng, userIDs, *minItem) {
			action := actions[rng.Intn(len(actions))]
			ts := time.Now().Add(-time.Duration(rng.Intn(7*24*3600)) * time.Second).Unix()
			behOut = append(behOut, []any{uid, rid, action, ts})
			if action == "favorite" && rng.Intn(2) == 0 {
				ratOut = append(ratOut, []any{uid, rid, 1 + rng.Intn(5), ts})
			}
		}
	}
	nBeh := insertIgnore(db, "user_behaviors",
		[]string{"user_id", "resource_id", "action", "created_at"},
		behOut, 2000, "FROM_UNIXTIME")
	nRat := insertIgnore(db, "ratings",
		[]string{"user_id", "resource_id", "score", "created_at"},
		ratOut, 2000, "FROM_UNIXTIME")

	nAdmin := 0
	if *admin {
		nAdmin = ensureAdmin(db, string(hash))
	}

	fmt.Printf("[mock_users] 用户=%d 行为=%d 评分=%d 管理员ID=%d（复用现有资源 %d 条）\n",
		len(userIDs), nBeh, nRat, nAdmin, len(resIDs))
	fmt.Printf("[mock_users] 账号: mock1~mock%d / %s\n", len(userIDs), *password)
}

// pickN 从切片随机选 n 个不重复元素（不足 n 则全选）。
func pickN(rng *rand.Rand, ids []uint, n int) []uint {
	if n >= len(ids) {
		return ids
	}
	perm := rng.Perm(len(ids))
	out := make([]uint, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, ids[perm[i]])
	}
	return out
}

// insertIgnore 分批 INSERT IGNORE；tsMode=FROM_UNIXTIME 时最后一列时间参数
// 用 FROM_UNIXTIME(?) 包裹。返回受影响行数（含忽略）。
func insertIgnore(db *gorm.DB, table string, cols []string, rows [][]any,
	chunk int, tsMode ...string) int {
	if len(rows) == 0 {
		return 0
	}
	useUnix := len(tsMode) > 0 && tsMode[0] == "FROM_UNIXTIME"
	colStr := "`" + strings.Join(cols, "`,`") + "`"
	var n int
	for start := 0; start < len(rows); start += chunk {
		end := start + chunk
		if end > len(rows) {
			end = len(rows)
		}
		part := rows[start:end]
		args := make([]any, 0, len(part)*len(cols))
		valueStrs := make([]string, 0, len(part))
		for _, row := range part {
			marks := make([]string, len(cols))
			for ci, v := range row {
				if useUnix && ci == len(cols)-1 {
					marks[ci] = "FROM_UNIXTIME(?)"
				} else {
					marks[ci] = "?"
				}
				args = append(args, v)
			}
			valueStrs = append(valueStrs, "("+strings.Join(marks, ",")+")")
		}
		sql := fmt.Sprintf("INSERT IGNORE INTO %s (%s) VALUES %s",
			table, colStr, strings.Join(valueStrs, ","))
		if err := db.Exec(sql, args...).Error; err != nil {
			log.Fatalf("写入 %s 失败: %v", table, err)
		}
		n += len(part)
	}
	return n
}

// ensureAdmin 确保存在 demo_admin 并设为管理员。
func ensureAdmin(db *gorm.DB, passwordHash string) int {
	var id uint
	if err := db.Raw("SELECT id FROM users WHERE username = ? LIMIT 1", "demo_admin").Scan(&id).Error; err != nil || id == 0 {
		if err := db.Exec(
			"INSERT IGNORE INTO users (username, email, password_hash, display_name, created_at) VALUES (?, ?, ?, ?, ?)",
			"demo_admin", "demo_admin@sim.local", passwordHash, "演示管理员", time.Now()).Error; err != nil {
			log.Fatalf("创建 demo_admin 失败: %v", err)
		}
		_ = db.Raw("SELECT id FROM users WHERE username = ? LIMIT 1", "demo_admin").Scan(&id).Error
	}
	if err := db.Exec("INSERT IGNORE INTO admins (user_id) VALUES (?)", id).Error; err != nil {
		log.Fatalf("设置管理员失败: %v", err)
	}
	return int(id)
}
