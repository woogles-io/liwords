# API Calls

## Prior to tournament

### NewTournament

Called at the very beginning. Typically Woogles admins do this to create a tournament "page". Use type TOURNAMENT to use Woogles pairings.

### SetTournamentMetadata

Basically the sidebar info. Can do this with the Woogles admin panel.

### Set tournament controls

Commonly only used at the very beginning of a tournament.

Set all round controls for a tournament, i.e., set pairing systems to be used
on a round-by-round basis

- Call TournamentService.SetTournamentControls with the right parameters.

### AddDivision

Add a division to your tournament.

### RemoveDivision

self-explanatory

### AddPlayers

Add a list of players to a division

### RemovePlayers

remove players from a division

## During tournament

### StartTournament

Start the tournament!

- I think this was removed from the newer messages
- Just starts every division.

### Set single round controls

Set the controls (pairing systems) for a single round number. Typically we do this when we have to redo pairings for a future round (and they were different from what was originally put in, with `SetTournamentControls`)

- Call TournamentService.SetSingleRoundControls with the right parameters.
- Only call this for rounds that have not yet opened.

### PairRound

Re-pair a round. Only use this for rounds that have not yet opened. THIS CALL WILL
WIPE OUT RESULTS if called on a round number that has already started!!

### SetPairing

Pair two players in an unopened round!

# Common scenarios

- A person doesn't show up for a round

  We must give them a forfeit loss after some time and their opponent a bye
  We must give them forfeit losses for the rest of the tournament (remove player?)

- Pairings seem wrong

  SetPairing for all relevant players one by one
  Manually verify that all players are paired for that round

-
