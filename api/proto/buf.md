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

Note that *all* requests to the Woogles API are HTTP POSTs. This is a requirement of the framework we use, [Twirp](https://twitchtv.github.io/twirp/).

## Errors

If the API call returns a 200 status code it was successful. Any other code could be an error. The error text is returned in JSON format.

## Authentication

Not all API calls require authentication, but the ones that do will return an error.

To authenticate, you can either use a session cookie (the Woogles webapp does this), or use an API key. You can create API keys within the settings menu of woogles.io.

If using an API key, you must send it in the HTTP request header `X-Api-Key`.

Note: not every single endpoint takes in an API Key, but most do.

## Input and Output formats

Use `Content-Type: application/json` for JSON, or `Content-Type: application/protobuf` for Protobuf. 

It's much easier to use the API with JSON + a web browser, but the protobuf option is still available and we use protobuf communication/serialization internally.

Let's use as an example the `omgwords_service.GameEventService`, which you can see in the `omgwords_service/omgwords.proto` file below this directory.

We want to create a new game for broadcast. We look at the `rpc CreateBroadcastGame(CreateBroadcastGameRequest) returns (CreateBroadcastGameResponse)` function call. All supported RPCs (Remote Procedure Calls) are listed in a `service` block in a `.proto` file. 

In this case, the endpoint would be `https://woogles.io/twirp/omgwords_service.GameEventService/CreateBroadcastGame`.

You can obtain the `omgwords_service` in the URL from the package name (see the top of the `omgwords.proto` file). `GameEventService` is the name of the rpc service. And `CreateBroadcastGame` is the function name. The format of the URL is `/twirp/<package_name>.<service_name>/<rpc_name>`.

We see that `CreateBroadcastName` takes in a `CreateBroadcastGameRequest`:

```proto
message CreateBroadcastGameRequest {
  repeated ipc.PlayerInfo players_info = 1;
  string lexicon = 2;
  ipc.GameRules rules = 3;
  ipc.ChallengeRule challenge_rule = 4;
  bool public = 5;
}
```

So we can create a game with the following JSON payload:

```json
{
    "players_info": [{
        "user_id": "user1", "nickname": "john", "full_name": "John Doe", "first": true
    }, {
        "user_id": "user2", "nickname": "jane", "full_name": "Jane Doe"
    }],
    "lexicon": "CSW21",
    "rules": {
        "board_layout_name": "CrosswordGame",
        "letter_distribution_name": "english",
        "variant_name": "classic"
    },
    "challenge_rule": "ChallengeRule_FIVE_POINT",
}
```

This endpoint will return a `CreateBroadcastGameResponse`; the json will look like:

```json
{
    "game_id": "abcdefg"
}
```

You can then use this game ID for future calls to this endpoint.

## Specific types

Protobuf types map to JSON in a pretty obvious way, but there are a couple of types that require special attention:

- int64 or uint64 are encoded as strings
- The `bytes` type is represented in JSON as a base64-encoded string. For example, if you wish to use the `SetRacksEvent` in `omgwords_service/omgwords.proto`, it uses a field `repeated bytes racks = 2;`. For that particular event, it accepts an array of byte arrays; the length of the outer array is the number of players in the game, the length of each byte array is the letters one wishes to assign the players.

If we wish to assign player 1 an empty, or unknown rack, and player 2 the rack "AEINST?", this SetRacksEvent must be sent to the backend as:

```json
{
    "game_id": "abcdefg",
    "racks": [
        "",
        "AQUJDhMUAA=="
    ]
}
```

`AQUJDhMUAA==` is the base64-encoding of the bytes `[1, 5, 9, 14, 19, 20, 0]` which correspond to the letters `AEINST?`. See the comments in SetRacksEvent in the `omgwords.proto` file referenced above.