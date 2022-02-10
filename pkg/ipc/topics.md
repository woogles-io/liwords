Each service should subscribe to one or more topics.

Examples:

Game Service subscribes to `gamesvc.>`

    - gamesvc.gameevt

Store Service subscribes to `storesvc.>`

    - storesvc.get
    - storesvc.put

The socket service is a bit special. At the moment it subscribes to stuff like:

- `game.>`
- `gametv.>`
- `chat.>`
- `lobby.>`

The reason for this is that the socket svc has a concept of "internal" subscriptions; i.e. a socket connection is subscribed to a "realm" named game-ABCDEFG. Then, when a message with topic game.ABCDEFG comes in, the socket server internally routes it to all internally subscribed sockets.

We should keep these subscriptions for the most part. When a game changes, the flow looks like this:

- Player makes a move in a game, gets sent via socket
  - i.e. GAME_EVT gets sent via protobuf from browser to socket (right now)
  - socket pushes that to gamesvc.gameevt.game_id or whatever
- One gamesvc picks it up, requests info from storesvc with req/resp
  - store.game.get or whatever, which publishes back to reply channel
- Game service plays the move and determines a new state
- Game service makes a req/resp SAVE request to the store service:
  - store.game.save or something similar
- Game service then publishes new game state to game.id, gametv.id

So, socketsvc can subscribe to game.id and gametv.id and publish the state back to any internal user subscriptions.

Analysis module can subscribe to game.id etc

req/resp requests should be retried if they time out (~2 sec timeout?). Use retry module.

---

Proposed channel names:

- socket still subscribes to game.>, gametv.>, etc
- socket now needs to know the message type so it can properly decode it, so the topics will look like:
  - game.pb.3.abcdef
  - user.pb.5.abcdefghijklmnop
- omgsvc would subscribe to omgsvc.>, so the topics look like:
  - omgsvc.pb.8.userid.auth.wsconnid
- storesvc is a little more freeform. it can subscribe to storesvc.> but topics look like `storesvc.omgwords.newgame`.
