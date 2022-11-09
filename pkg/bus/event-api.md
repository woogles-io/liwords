Bus should listen on `game.>` and `user.>` as well.

Event API stream should "subscribe" to a channel such as game.abcdef or 
user.uid.game.abcdef.
- It should unsubscribe when the http request is over.
- It should start a select loop in the http request
    - Listen on request-specific channel req.uuid 
    - Exit when channel is closed or similar?

When a message comes in on `game.abcdef` or `user.uid.game.abcdef`, we should push it to every subscription to it.
- Find the request-specific channel(s) req.uuid subscribed to this channel.
- Publish message to each of them.

### Data structures

Therefore we want to use data structures like this:

```go
var channelNamesForReqId map[string]string
var reqIDsForChannelName map[string][]string
var byteChanForReqId map[string]chan []byte
```

`reqIDsForChannelName` is a map of channel name (such as `game.abcdef`) to request ID. It can have many request IDs in a list

`byteChanForReqId` is a map of request ID to the chan that actually transfers the data across. The chan actually should receive the message and transfer it to the user.

API serve function should listen to its specific req ID chan for data.
When it is done it should delete it from `byteChanForReqID` (easy) and from the `reqIDsForChannelName` map.