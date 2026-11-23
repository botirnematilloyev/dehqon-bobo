#!/usr/bin/env python3
"""locations.csv -> migratsiya SQL (upsert, remote_id bo'yicha idempotent).

    python3 scripts/geo/generate.py migrations/000014_sync_locations.up.sql

Mavjud yozuvlar nomi yangilanadi, yangilari qo'shiladi; hech narsa
o'chirilmaydi (foydalanuvchi va xo'jaliklar district_id orqali bog'langan).
"Chet el" (9999) — dehqon ilovasi uchun kerak emas, olinmaydi.
"""
import csv
import json
import pathlib
import sys

SRC = pathlib.Path(__file__).with_name("locations.csv")
SKIP = {"9999"}


def lit(s: str) -> str:
    return "'" + s.replace("'", "''") + "'"


def name(r) -> str:
    return lit(json.dumps({"uz": r["name_uz"], "uz_cyr": r["name_uz_cyr"], "ru": r["name_ru"]},
                          ensure_ascii=False, separators=(",", ":")))


def main(out: str):
    rows = list(csv.DictReader(SRC.open(encoding="utf-8")))
    regions = [r for r in rows if r["kind"] == "region" and r["remote_id"] not in SKIP]
    districts = [r for r in rows if r["kind"] == "district" and r["parent_remote_id"] not in SKIP]
    sql = [
        "-- scripts/geo/locations.csv dan generate.py bilan yasalgan (manba: pm.gov.uz).",
        "-- QO'LDA TAHRIR QILINMAYDI. remote_id bo'yicha upsert — qayta ishlatish xavfsiz.",
        "",
        "INSERT INTO regions (remote_id, name) VALUES",
        ",\n".join(f"    ({lit(r['remote_id'])}, {name(r)})" for r in regions),
        "ON CONFLICT (remote_id) DO UPDATE SET name = EXCLUDED.name;",
        "",
        "INSERT INTO districts (remote_id, region_id, name)",
        "SELECT d.remote_id, r.id, d.name::jsonb",
        "FROM (VALUES",
        ",\n".join(f"    ({lit(r['remote_id'])}, {lit(r['parent_remote_id'])}, {name(r)})" for r in districts),
        ") AS d(remote_id, region_remote_id, name)",
        "JOIN regions r ON r.remote_id = d.region_remote_id",
        "ON CONFLICT (remote_id) DO UPDATE SET name = EXCLUDED.name, region_id = EXCLUDED.region_id;",
        "",
    ]
    pathlib.Path(out).write_text("\n".join(sql), encoding="utf-8")
    print(f"{len(regions)} regions, {len(districts)} districts -> {out}")


if __name__ == "__main__":
    main(sys.argv[1])
