To run this script with Docker, cd up to the main `liwords` directory and run:

```
docker compose run --rm -w /opt/program/cmd/puzzlegen app go run . '{"bot_vs_bot":true,"lexicon":"NWL20","letter_distribution":"english","sql_offset":0,"game_consideration_limit":1000000,"game_creation_limit":200,"request":{"buckets":[{"size":25,"includes":["EQUITY","CEL_ONLY","NON_BINGO"],"excludes":[]},{"size":25,"includes":["EQUITY","BINGO"]},{"size":25,"includes":["EQUITY","NON_BINGO"]},{"size":25,"includes":["EQUITY","BINGO","CEL_ONLY"]}]}}'
```

See the PuzzleGenerationJobRequest in puzzle_service.proto for example format of the JSON.

```
docker compose run --rm -w /opt/program/cmd/puzzlegen app go run . '{"bot_vs_bot":false,"lexicon":"CSW21","letter_distribution":"english","game_consideration_limit":20,"request":{"buckets":[{"size":5, "includes":["EQUITY"]}]}}'
```