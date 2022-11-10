// in charge of creating game events locally for the board editor.

import { Action, ActionType } from '../actions/actions';
import { ServerGameplayEvent } from '../gen/api/proto/ipc/omgwords_pb';
import { ClientGameplayEvent, ClientGameplayEvent_EventType } from '../gen/api/proto/ipc/omgwords_pb';

export const eventDispatcher = (
  evt: ClientGameplayEvent,
  dispatchGameContext: (action: Action) => void
) => {

  // convert the event into a ServerGameplayEvent.


  switch (evt.type) {
    case ClientGameplayEvent_EventType.TILE_PLACEMENT:


      dispatchGameContext({
        actionType: ActionType.AddGameEvent,
        payload: new ServerGameplayEvent({
          event: 
        })
      })

  }


};


const toServerGameplayEvent = (evt :ClientGameplayEvent): ServerGameplayEvent => {

  switch (evt.type) {
    case ClientGameplayEvent_EventType.TILE_PLACEMENT:
      
  }
}