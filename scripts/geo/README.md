# Viloyat va tumanlar

Manba: `pm.gov.uz:8020/dictionary/regions` (murojaat.gov.uz). Yangilash:

```bash
python3 scripts/geo/fetch.py                                          # -> locations.csv
python3 scripts/geo/generate.py migrations/<N>_sync_locations.up.sql  # CSV -> SQL (upsert)
```

`locations.csv` repoda saqlanadi — migratsiya qaysi ma'lumotdan yasalgani ko'rinib tursin.
Upsert `remote_id` bo'yicha: nomlar yangilanadi, yangilari qo'shiladi, hech narsa o'chirilmaydi.
