// A new game reducer based on immer and the GameDocument structure.
import produce from 'immer';
import { Action, ActionType } from '../../actions/actions';
import { Alphabet } from '../../constants/alphabets';
import {
  GameDocument,
  GameEvent,
  ServerGameplayEvent,
} from '../../gen/api/proto/ipc/omgwords_pb';
import { GameEvent as MacondoGameEvent } from '../../gen/api/proto/macondo/macondo_pb';
import { PlayedTiles, PlayerOfTiles } from '../../utils/cwgame/common';
import { PlayerOrder } from '../constants';
import { ClockController, Millis } from '../timer_controller';

export type GameState = {
  document: GameDocument;
  alphabet: Alphabet;
  lastPlayedTiles: PlayedTiles;
  playerOfTileAt: PlayerOfTiles;
  clockController: React.MutableRefObject<ClockController | null> | null;
  onClockTick: (p: PlayerOrder, t: Millis) => void;
  onClockTimeout: (p: PlayerOrder) => void;
};

// export const startingGameState = (
//   players: Array<
// )

const macondoEventToOMGWordsEvent = (
  mevt: MacondoGameEvent,
  alphabet: Alphabet
): GameEvent => {
  return new GameEvent({
    rack: runesToUint8Array(mevt.rack, alphabet),
    type: mevt.type.valueOf(),
    cumulative: mevt.cumulative,
    row: mevt.row,
    column: mevt.column,
  });
};

export const GameReducer = (state: GameState, action: Action): GameState => {
  switch (action.actionType) {
    // case ActionType.ClearHistory: {

    // }
    case ActionType.AddGameEvent: {
      const sge = action.payload as ServerGameplayEvent;
      if (sge.gameId !== state.document.uid) {
        return state;
      }
      if (!sge.event) {
        return state;
      }
      const nextState = produce(state, (draftState) => {
        draftState.document.events.push(sge.event!);
      });
      // const ngs = newGameStateFromGameplayEvent(state, sge);
      // ngs.clockController = state.clockController;
      // setClock(ngs, sge);
      // return ngs;
      return nextState;
    }
  }
};
