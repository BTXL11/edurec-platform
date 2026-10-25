"""单条视频关键帧采集入口（供 Go 后端 os/exec 调用，也供离线批处理复用）。

用法：

    python frames.py --bvid BV1DgxCzREbM

约定（Go 侧 exec.Command().Output() 依赖）：

- stdout 只输出一行 JSON：
    {"bvid": "...", "cover_url": "...", "frames": {...}}
- 日志与错误一律走 stderr；失败返回非 0 退出码、stdout 为空
- 编码统一 UTF-8（Windows 子进程默认 GBK 会乱码）

关键帧来自 B 站公开接口 /x/player/videoshot（进度条预览雪碧图），
无需登录、不下载视频，只保存图片 URL 与网格参数。
"""

from __future__ import annotations

import argparse
import json
import sys

from bili_client import BiliClient, BiliError, RiskControlError
from collect import _normalize_cover, normalize_videoshot


def _force_utf8() -> None:
    for stream in (sys.stdout, sys.stderr):
        if hasattr(stream, "reconfigure"):
            try:
                stream.reconfigure(encoding="utf-8", errors="replace")
            except (ValueError, OSError):
                pass


def _emit(payload: dict) -> None:
    body = json.dumps(payload, ensure_ascii=False)
    sys.stdout.buffer.write(body.encode("utf-8") + b"\n")
    sys.stdout.buffer.flush()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="采集 B 站视频关键帧")
    parser.add_argument("--bvid", required=True, help="视频 BV 号")
    parser.add_argument("--cookie", default="", help="可选：自定义 cookie")
    parser.add_argument("--timeout", type=int, default=15, help="单次请求超时秒数")
    return parser.parse_args()


def main() -> int:
    _force_utf8()
    args = parse_args()

    client = BiliClient(
        cookie=args.cookie,
        delay=(0.4, 0.8),
        max_retries=2,
        timeout=args.timeout,
        verbose=False,
    )
    try:
        # 详情用于补封面（资源可能缺封面）；关键帧接口单独调用
        detail = client.video_detail(args.bvid)
        shots = client.video_shots(args.bvid)
    except RiskControlError as exc:
        print(f"[frames] 被 B 站风控拦截：{exc.message}", file=sys.stderr)
        return 2
    except BiliError as exc:
        print(f"[frames] 接口调用失败：{exc}", file=sys.stderr)
        return 1
    except Exception as exc:  # 网络异常等，统一按失败处理
        print(f"[frames] 采集失败：{exc}", file=sys.stderr)
        return 1
    finally:
        client.close()

    frames = normalize_videoshot(shots, args.bvid)
    payload = {
        "bvid": args.bvid,
        "cover_url": _normalize_cover(detail.get("pic", "")),
        "frames": frames,
    }
    if not frames["sprites"]:
        print("[frames] 该视频没有可用的关键帧（可能过短或未生成）", file=sys.stderr)
        payload["frames"]["sprites"] = []

    _emit(payload)
    return 0


if __name__ == "__main__":
    sys.exit(main())
