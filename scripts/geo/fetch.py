#!/usr/bin/env python3
"""Viloyat va tuman/shaharlarni pm.gov.uz (murojaat.gov.uz manbasi) dan bir
marta tortib, locations.csv ga yozadi.

    python3 scripts/geo/fetch.py            # -> scripts/geo/locations.csv

Manba tillari (`language` sarlavhasi): oz — o'zbek lotin, uz — o'zbek kirill,
ru — rus. CSV ustunlari:

    kind,remote_id,parent_remote_id,name_uz,name_uz_cyr,name_ru

Migratsiya SQL'i shu CSV'dan generate.py bilan yasaladi.
"""
import csv
import json
import pathlib
import sys
import time
import urllib.request

BASE = "https://pm.gov.uz:8020/dictionary/regions"
LANGS = {"oz": "name_uz", "uz": "name_uz_cyr", "ru": "name_ru"}
OUT = pathlib.Path(__file__).with_name("locations.csv")


def get(parent, lang):
    url = BASE + (f"?parent_id={parent}" if parent else "")
    req = urllib.request.Request(url, headers={"Accept": "application/json", "language": lang,
                                               "User-Agent": "dehqon-bobo-geo/1.0"})
    for attempt in range(3):
        try:
            with urllib.request.urlopen(req, timeout=30) as r:
                return json.load(r)
        except Exception as e:  # tarmoq xatosi — qayta urinish
            if attempt == 2:
                raise
            print(f"retry {url} {lang}: {e}", file=sys.stderr)
            time.sleep(2)


def collect(parent):
    rows = {}
    for lang, col in LANGS.items():
        for item in get(parent, lang):
            rows.setdefault(item["id"], {})[col] = item["title"].strip()
    return rows


def main():
    regions = collect(None)
    out = []
    for rid, names in regions.items():
        out.append({"kind": "region", "remote_id": rid, "parent_remote_id": "", **names})
        for did, dnames in collect(rid).items():
            out.append({"kind": "district", "remote_id": did, "parent_remote_id": rid, **dnames})
        time.sleep(0.3)  # manbaga yumshoq
    for r in out:
        for col in LANGS.values():
            if not r.get(col):
                sys.exit(f"missing {col} for {r['kind']} {r['remote_id']}")
    with OUT.open("w", newline="", encoding="utf-8") as f:
        w = csv.DictWriter(f, fieldnames=["kind", "remote_id", "parent_remote_id", *LANGS.values()])
        w.writeheader()
        w.writerows(out)
    print(f"{sum(r['kind'] == 'region' for r in out)} regions, "
          f"{sum(r['kind'] == 'district' for r in out)} districts -> {OUT}")


if __name__ == "__main__":
    main()
