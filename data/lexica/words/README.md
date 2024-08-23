If `../gaddag/CSW19.kwg` exists, put `CSW19.txt` here (`data/lexica/words`) and `docker compose restart app`.

The format is tab-separated, one definition per line, `WORD\tdefinition\n` and it must be sorted with C collation (`LC_ALL=C sort`).

You can generate it from a Zyzzyva db file, for example, as follows:

```
$ cat makedict.sh
#!/usr/bin/env bash
sqlite3 -separator $'\t' ~/.collinszyzzyva/lexicons/CSW19.db "select word, replace(definition, x'0A', ' / ') as definition from words" | LC_ALL=C sort > CSW19.txt
```