See arch.dot

Since this is a real-time game, we don't need sub millisecond times.

A circle with NATS in the middle.

Each "service"/module communicates through NATS. Each module is easily testable, deployable, and independent. Each only has inputs/outputs

Communication between services and NATS should be protobuf. Communication to user doesn't need to be protobuf. Socket comms can be plain text with a translation later. Twirp API can maybe be replaced with something more RESTful if desired. See OpenAPI.

### Sample requests

#### Make a move in a game
- Player makes a move in a game; gets sent via socket
- Write GAME_EVT to NATS conn with game ID and information about the game. (Turn/evt number should be included too, as a good way to ensure the sequence is correct).
- NATS publishes to listening game service(s) 
    - We can use a "Queue group" so that only one game service takes the request
- Game service sends a request (via NATS) for the game information (use request/response with a queue group)
- Store service picks up the request, sends game information back as protobuf
- Game service deserializes it and plays the game to the current turn, and then plays the new turn.
    - Deserialization should be optimized somewhat. Even right now it is very fast, HastyBot does this every turn.
- Game service publishes state back to NATS
    - e.g. GAME_STATE_MSG, GAME_ID, full pb, last turn pb
- Any listeners/subscribers can do the needful:
    - The db can save the state back, and if the game is over, it can save it to S3
        - Save game + new user ratings in a transaction
    - The socket server can publish just the last move (or maybe the full pb might simplify the front end)
    - Anti-cheating module can be listening for finished games and do whatever analysis on them
    - Tourney module can be listening for an ended game and update it
    - Automod/other module can be listening and apply penalties to users who do bad things
    - etc


#### Request last games in a profile
- Twirp API module picks up request, sends a message via NATS using req-response
- Store module picks up the request, makes the necessary DB requests and sends it back through protobuf
- Twirp API module receives answer, does any necessary conversions to send back data to user.


### Start a game
- A seek/match is answered, or a tournament game is ready
- Main API module (or whoever gets the signal? tournament module?) sends a message to gamesvc via PB
    - user IDs/usernames
    - req (GameRequest)
    - assignedFirst
    - tournament data, if any
- gamesvc tells store (req/resp) to save a new game with these parameters, and gets ID back
    - use retry module with a short ID
- gamesvc publishes back details of new game (ID) to both participants