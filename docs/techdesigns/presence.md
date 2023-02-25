# Technical Design for Presence

Date: 2/24/23

Authors: César Del Solar

# Overview #

We wish to implement presence in an efficient way. User presence answers questions like the following:

- Which of the people that I am following are online?
- What are these people currently doing? Playing games? Analyzing? Solving puzzles?
- When is the last time that the person I am following logged on?
- How many other people are currently in this chat? Even if I am not following them, I should be able to see this information.

Right now, presence is implemented in a processor and IO intensive way across multiple Redis keys and NATS subscriptions; it is difficult to debug, does not have obvious data structures, and involves complex in-Redis Lua scripts.

We propose that this can be done correctly from scratch with a more persistent data store, and using efficient patterns.

## socketsvc ##

The source of truth for user presence is `liwords-socket` (https://github.com/domino14/liwords-socket). We can also refer to this service as `socketsvc`.

`socketsvc` handles all client Websocket connections. A client subscribes to a "realm". Currently, a client logs in, and `socketsvc` asks `liwords-api` which realm the client should be subscribed to. Some examples of realms are `game-abcdef`, `gametv-defghi`, etc. `liwords-api` needs to determine the realm by looking in the database for which players are involved in a game, so that the user does not request a realm they should not have access to.

`socketsvc` and `liwords-api` communicate through a NATS server. Whenever `liwords-api` sends a message on a channel with a pattern like `user.userID`, every socketsvc gets this message, and then looks in its connection list to figure out which user to deliver this message to.

`socketsvc` does not currently contain any connections to a database. This was done to keep the service as simple as possible.

## liwords-api ##

`liwords-api` is the main API server (this repo). It handles pretty much everything about liwords except for the socket connections.

When a player performs an action, like starting a game, ending a game, joining a "realm" (such as puzzles), we look up all their followers, and send a NATS message to `user.<UserID>` for each one of their followers.

This is inefficient and results in large amounts of identical messages being sent on a buffered channel.  NATS doesn't like this very much.


# Implementation #

In order to make things more scalable, and to avoid sending so many messages on NATS, we propose to be less zealous about `liwords-socket` having no database access. It would simplify a lot of flows a great deal if `liwords-socket` could talk to the Postgres user database. We still want to limit the total surface area in terms of which tables it can talk to, but overall we should be able to keep things maintainable without making the code too complex.


## New database tables

We need a table to hold user presences.

```sql
CREATE TABLE public.user_presences (
    user_id int,
    last_seen timestamp with time zone
);

CREATE UNIQUE INDEX idx_user_presences_id ON public.user_presences USING btree(user_id);
```

We need another table for channel presences:

```sql
CREATE TABLE public.channel_presences (
    channel_name string,
    last_updated timestamp with time zone
);

CREATE UNIQUE INDEX idx_channel_presences_channel ON public.channel_presences USING btree(channel_name);
```

We need a table to tie the two together:

```sql
CREATE TABLE public.user_channel_presences (
    user_id int,
    channel_name string,
    connection_id string
);

CREATE UNIQUE INDEX idx_user_channel_presences ON public.user_channel_presences USING btree(user_id, channel_name, connection_id);
CREATE INDEX idx_uc_presences_connid ON public.user_channel_presences USING btree(connection_id);
```


## liwords-socket patterns

### Login

When a user logs in to the socket, we will subscribe them to their regular channels/realms as we do now.

We will modify the `user_presences` (and perhaps other presences tables) in a transaction, adding this connection ID and to the relevant channel.

**Note**: For legacy purposes the user connection ID is not of full UUID length. We should either lengthen it, or handle the rare collision case by prompting the user to log in again.

The user also needs to get information about what the people they follow are up to. We will make a batch query to the `public.user_channel_presences` table for every user_id they follow, and send this data out via the socket as well.

Finally, the user needs information on the channel they just joined. We make a query to `public.user_channel_presences` table and get the list of users also in this channel. This can get sent out via the socket as well.


We can then send a message on NATS with the channel `followersof.<UserID>` and the user's presence information (lobby? tournament room? etc).

The socket listens on `followersof.>`. Note that we can have multiple socket servers, so each socket server will get this message. 

Upon receipt of a `followersof.>` message:

- We will then do a database lookup for users who follow the relevant user. We can keep an in-memory LRU cache with a low expiry time for this (a few seconds), as it's possible for users to refresh rapidly, and we don't want to do too many of these requests. With the right tuning, however, Postgres could have its own cache for this data.
- We can then send a socket message to every user in this list that is currently online (according to this particular `socketsvc` node).

**Note**: We currently login to the socket every time we switch pages in the app, very quickly after logging out from a previous session. 



### Logout

When a user logs out of the socket, we can send a NATS message to `followersof.<UserID>` as in the above flow. 

We also do the reverse of the above -- modify the `user_presences` and other tables to remove the player's connection ID.

If a user is switching pages, they rapidly log out and in again to another "realm". They can also log out of one `socketsvc` node and log in to another one.

There doesn't appear to be an easy way to "consolidate" messages here. The LRU cache can help here, though, in getting the same list of user IDs a fraction of a second apart for a user, if they log in to the same socket.

**Note**: The proper solution here is to modify the socket protocol and app to minimize the number of socket connections/disconnections when navigating to different pages. But this is outside of the scope of this doc.



### Multiple login

A user can be logged into the socket multiple times (on different tabs). 

On every login, we just add more rows to the `user_channel_presences` table with different connection IDs.

### Ping

`socketsrv` sends a websocket ping for every connection every few seconds, and then reads a pong back from the client.

Every few pongs, we should update the `last_seen` column for this user.


## liwords-api patterns

### A user is followed

When a user is followed in liwords-api, the follower would like to see more information about what the followee is doing. `liwords-api` should send a NATS message to `user.<FollowerID>` with the channels that followee is in.

### A user is unfollowed

Here we can just modify the followers table and keep current behavior. Front end should hide any information about followee.

### A game starts/ends

When a game starts:

- Add involved users to a channel `activegame#<GameID>` in the database. Note that they should already be in the channel `game#<GameID>` because they're in the "table". Now they'll be in both channels, with the same connection ID.

When a game ends:

- Remove involved users from channel `activegame#<GameID>`.

For both cases:

- Send a message to `followersof.<UserID>` with the new information.

### User starts a puzzle

- Add user to `lobby#puzzles`
- Send a message to `followersof.<UserID>`

### User starts annotating a game

- Add user to `anno#<GameID>`
- Send a message to `followersof.<UserID>`

## Cronjob

We need a cronjob to keep the table clean. It is possible to run into a situation where a presence is stale (in the database but not actually connected to the socket). We can run a daily or hourly job to clear all presences where `last_seen` is older than some amount of time.

Note that we should always update the `user_presences` and `channel_presences` together, in a transaction. That way, there is no chance those two are out of sync with each other. When the cronjob deletes from the `user_presences` table, then, it should find every channel that user is in, and delete the user from the relevant `channel_presences` item. Of course, this should also be done in a transaction.

The cronjob should be a periodic ECS task.

## Data structures

### Messages

We should make a UserActivityType proto:

```proto
enum UserActivityType {
    TypeNone = 0;
    TypeDisconnected = 1;
    TypeStartedGame = 2;
    TypeEndedGame = 3;
    TypeSolvingPuzzles = 4;
    TypeAnnotatingGame = 5;
    TypeInGameChannel = 6;
}
```

We send messages to a NATS channel `followersof.<UserID>`. The message contents should be a protobuf like:

```proto
message FolloweeActivity {
    ActivityType atype = 1;
    string user_id = 2;
    // meta can likely be the name of the channel or some other message.
    string meta = 3;
}
```

**Note**: receivers of these messages may get multiple conflicting messages. For example, a player can be doing puzzles, then they can get an EndedGame message, because they were playing a game in another tab. It is up to the receiver to decide how to depict this. Typically the receiver is the front-end and it can prioritize which `UserActivityType`s to show over others.


### Channels

Our channels are for presence. These are not the same channels as the internal NATS channels that we use for inter-process communication. To differentiate, we can format them in a different way.

For separating logical parts of a presence channel, we can use a pleasing symbol not in the base64 set, such as #.

`lobby#omgwords`
`game#ABCDEF`
`gametv#ABCDEF`
`lobby#puzzles`
`anno#ABCDEF`
`activegame#ABCDEF`
`lobby#tournament#ABCDEF`