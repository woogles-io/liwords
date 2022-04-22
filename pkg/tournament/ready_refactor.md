Proposal:

Remove Tournament Round Ready message.

We already have a Ready signal that gets sent in the actual game. Having two adds more chances for failure, but most importantly, our current implementation of it is very finiking and takes up a lot of cpu/memory from repeated serialization and deserialization of a large JSONB blob.

SetReadyForGame should:
- Immediately start a new game if you're the first one to click it, and redirect you to the game [UX Change]:
    - Edit the pairing in the tournament to add the game ID to the `games` column

    ```proto
    message TournamentGame {
    repeated int32 scores = 1;
    repeated TournamentGameResult results = 2;
    GameEndReason game_end_reason = 3;
    string id = 4;
    }

    message Pairing {
    repeated int32 players = 1;
    int32 round = 2;
    repeated TournamentGame games = 3; // can be a list, for elimination tourneys
    repeated TournamentGameResult outcomes = 4;
    repeated string ready_states = 5 [deprecated = true];
    }
    ```


    - Save this back to the database. Explore using a JSON operation
directly to minimize load.
    - Send a signal back to the frontend that someone clicked this. This should change the state for the second person (your opp is waiting for you).
- Join the already existing game if you're the second one to click it.

If you refresh the page after your opponent clicks Ready, we don't have any indication that the game hasn't yet started, and we probably don't need one. The widget could just default to saying what it says when you are in a tournament game but found yourself back in the lobby. [UX Change]

If you then click to join the game, this starts the game for real (both players are ready).
- This switches the chat to the player tab from the tournament tab.

- While the first player is waiting for the game to start, the chat should default to the tournament chat, so that they can keep abreast of any announcements.
    - The game board panel etc should not look broken. It should clearly say waiting for Opponent Name to click Ready.
    - The Ready widget should count down. After 5 minutes (or whatever) it would give the first clicker a forfeit win.
    - The adjudicator on the backend should ignore any tournament games for at least the forfeit win timer.

- what if ready doesn't trigger? There are situations in which both players have joined a game and the game doesn't start. It happens rarely, but it still happens, particularly if one of them has a poor connection.
    - This is difficult to fix. Ideally, the person who tried to join the existing game is not seeing it load and will refresh after some time. They still might forfeit on time.
    - The UI could prompt them to refresh after a few seconds of trying without a socket connection.
    - The actual fix involves not reconnecting the socket on game load, but this is a significantly larger change.


- [Optional]: Instead of starting the forfeit timer when the first player clicks Ready, start it when the tournament round opens. We will need this functionality for automatic tournaments.
    - If neither player clicks the Ready button before the timer expires, they both get forfeit losses.

- A couple of directors have asked for the ability to tell when a player has clicked ready so they know who to chide. Now, all games will show up as soon as the first person clicks Ready, but it won't necessarily start. As long as the auto-forfeit functionality works, maybe it's not necessary to have this feature.
- We can have a feature that uses our `activegames` presence channel to tell Observers whether both of the players they're observing are actually present in the game. This could be added in the future.