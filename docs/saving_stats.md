make a table liststats

```
gameid playerid stat item prob
foo cesar bingos FOOTERS 123123
foo cesar challenged BLAHS 123
```

select \* from liststats where playerid = cesar order by gameid
