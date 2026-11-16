#!/usr/bin/env bash
# HPG manbasini pin qilingan commit bo'yicha yuklab, golden JSON'ni qayta yaratadi.
#
#   scripts/hpg-golden/fetch.sh            # golden.json'dagi commit
#   scripts/hpg-golden/fetch.sh <sha>      # boshqa commit (yangilash)
#
# hpg.html va .hpg misollar repoga KIRITILMAYDI (GPLv3): ular vaqtinchalik
# katalogga tushadi. Natija: internal/modules/calc/hpg/testdata/golden.json
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUT="$ROOT/internal/modules/calc/hpg/testdata/golden.json"
REPO="WEGA-project/wega-hpg"

SHA="${1:-}"
if [[ -z "$SHA" ]]; then
  SHA="$(node -e "console.log(require('$OUT').commit)")"
fi
[[ "$SHA" =~ ^[0-9a-f]{40}$ ]] || { echo "need a full 40-char commit sha" >&2; exit 1; }

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
mkdir -p "$WORK/ex"

RAW="https://raw.githubusercontent.com/$REPO/$SHA"
curl -fsSL -o "$WORK/hpg.html" "$RAW/hpg.html"

# Barcha .hpg misollar (fayl nomida / → _).
curl -fsSL "https://api.github.com/repos/$REPO/git/trees/$SHA?recursive=1" |
  node -e '
    let s=""; process.stdin.on("data",d=>s+=d).on("end",()=>{
      // Kirill va "+" li nomlar bor: URL uchun kodlanadi, fayl nomi esa asl.
      for (const e of JSON.parse(s).tree) if (e.path.endsWith(".hpg"))
        console.log(e.path.split("/").map(encodeURIComponent).join("/") + "\t" + e.path);
    });' |
  while IFS=$'\t' read -r url p; do
    curl -fsSL -o "$WORK/ex/${p//\//_}" "$RAW/$url"
  done

node "$ROOT/scripts/hpg-golden/generate.mjs" "$WORK/hpg.html" "$WORK/ex" "$OUT" "$SHA"
echo "golden regenerated from $REPO@$SHA"
