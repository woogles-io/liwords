## liwords

This module contains all of the APIs required to interact with 
Woogles.io.

The base URL for all of the relevant services is:

`https://woogles.io/twirp/`

(This will change to /api soon, but use /twirp for now).

So for example to hit the game_service's GetGCG, you would do:

```
curl -H 'Content-Type: application/json' https://woogles.io/twirp/game_service.GameMetadataService/GetGCG -d '{"game_id": "abcdef"}'
```