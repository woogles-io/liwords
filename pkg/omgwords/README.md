This pkg is the foundation for a future new service - "omgwords-svc" - which handles omgwords gameplay. It is eventually meant to replace the pkg/gameplay module. It will use the pkg/cwgame package to implement the rules of the game.

omgwords-svc will do everything related to the gameplay that is not handled by the core rules pkg (cwgame).

For example:

- routing messages back and forth from cwgame to the rest of the app / message bus
- annotated game api
- gamedocument store