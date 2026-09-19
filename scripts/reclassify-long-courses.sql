-- =============================================================================
-- 把「长合集」类 B 站内容归入 course 类型
--
-- 背景：B 站采集统一按 type=video 落库（BilibiliImportOptions.ResourceType），
-- 于是资源库里全是 video —— 长课程、系列合集与十几分钟的单个视频混在一起，
-- 按类型筛选时无法区分。这里把「明显的长合集 / 系统课程」改判为 course。
--
-- 判定口径（与服务端 admin 的资源编辑接口等价，纯数据迁移，不改代码）：
--   标题含  合集 / 全集 / 全套 / 系列 / 全N集 / N集 / 课程 / 教程
--   且时长 ≥ 120 分钟（duration 形如 "2809:53"，即 分钟:秒）
--
-- 用法：
--   mysql -h 127.0.0.1 -P 3308 -u root -p edurec < scripts/reclassify-long-courses.sql
--
-- 该口径在 2026-09-19 的库上命中 43 条，明细（id: 时长分钟）留档如下，可据此回滚核对：
--   47:9005 57:9003 570:7597 569:5752 560:4578 619:4197 39:3535 44:3535 95:3397 69:3397
--   100:3357 62:3317 20:3259 10:2955 27:2814 18:2809 26:2578 4:2316 138:2314 53:1993
--   9:1923 151:1913 14:1891 140:1849 555:1603 5:1552 134:1503 115:1453 12:1419 2:1384
--   6:1009 126:991 104:985 1:968 610:936 13:934 3:747 23:634 564:572 612:565
--   38:276 110:250 152:121
--
-- 可反复执行（幂等：只改 type，已是 course 的行不受影响）；
-- 采集重复导入不会把它改回 video —— 判重命中时只刷新 view_count 与 metadata，
-- 不覆盖 type（见 internal/service/crawl_import.go 的 ImportItems）。
-- =============================================================================

-- ① 先看会改哪些行（建议先跑这一步确认）
SELECT id,
       CAST(SUBSTRING_INDEX(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.duration')), ':', 1) AS UNSIGNED) AS minutes,
       LEFT(title, 50) AS title
FROM resources
WHERE (title REGEXP '合集|全集|全套|系列|全[0-9]+集|[0-9]+集|课程|教程')
  AND CAST(SUBSTRING_INDEX(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.duration')), ':', 1) AS UNSIGNED) >= 120
ORDER BY minutes DESC;

-- ② 执行改判
UPDATE resources
SET type = 'course'
WHERE (title REGEXP '合集|全集|全套|系列|全[0-9]+集|[0-9]+集|课程|教程')
  AND CAST(SUBSTRING_INDEX(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.duration')), ':', 1) AS UNSIGNED) >= 120;

-- ③ 核对结果：应为 course 43 / video 162（2026-09-19 库规模）
SELECT type, COUNT(*) AS n FROM resources GROUP BY type;

-- 回滚：把本次改判的行改回 video（用同样的口径即可，因为改判只动了 type）
-- UPDATE resources
-- SET type = 'video'
-- WHERE type = 'course'
--   AND (title REGEXP '合集|全集|全套|系列|全[0-9]+集|[0-9]+集|课程|教程')
--   AND CAST(SUBSTRING_INDEX(JSON_UNQUOTE(JSON_EXTRACT(metadata, '$.duration')), ':', 1) AS UNSIGNED) >= 120;
