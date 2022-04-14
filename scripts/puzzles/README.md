To run this script with Docker, cd up to the main `liwords` directory and run:

```
docker-compose run --rm -w /opt/program/scripts/puzzles app go run . '{"bot_vs_bot":true,"lexicon":"NWL20","letter_distribution":"english","sql_offset":0,"game_consideration_limit":1000000,"game_creation_limit":200,"request":{"buckets":[{"size":25,"includes":["EQUITY","CEL_ONLY","NON_BINGO"],"excludes":[]},{"size":25,"includes":["EQUITY","BINGO"]},{"size":25,"includes":["EQUITY","NON_BINGO"]},{"size":25,"includes":["EQUITY","BINGO","CEL_ONLY"]}]}}'
```

Replace the flags after the `.` with whatever flags you want. The flags in the example above run 5 bot v bot games.
