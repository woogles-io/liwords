import React from 'react';
import { act, cleanup, render } from '@testing-library/react';
import { BoardPanel } from './board_panel';
import { ChallengeRule } from '../gen/api/proto/macondo/macondo_pb';
import { CrosswordGameGridLayout } from '../constants/board_layout';
import { Board } from '../utils/cwgame/board';
import { PlayerInfo } from '../gen/api/proto/ipc/omgwords_pb';
import { StandardEnglishAlphabet } from '../constants/alphabets';
import { BrowserRouter } from 'react-router-dom';
import { waitFor } from '@testing-library/react';

function renderBoardPanel() {
  const dummyFunction = () => {};

  const rack = [0, 1, 5, 9, 14, 19, 20];
  const board = new Board(CrosswordGameGridLayout);
  const playerInfo = [
    new PlayerInfo({
      userId: 'cesarid',
      nickname: 'cesar',
      fullName: 'cesar richards',
    }),
    new PlayerInfo({
      userId: 'oppid',
      nickname: 'opp',
      fullName: 'opp mcOppface',
    }),
  ];
  return render(
    <BrowserRouter>
      <BoardPanel
        anonymousViewer={false}
        username="cesar"
        currentRack={rack}
        events={[]}
        gameID={'abcdef'}
        challengeRule={ChallengeRule.DOUBLE}
        board={board}
        sendSocketMsg={() => {}}
        sendGameplayEvent={() => {}}
        gameDone={false}
        playerMeta={playerInfo}
        lexicon="NWL20"
        alphabet={StandardEnglishAlphabet}
        handleAcceptRematch={dummyFunction}
        handleAcceptAbort={dummyFunction}
        vsBot={false}
      />
    </BrowserRouter>
  );
}

afterEach(cleanup);

it('renders a game board panel', async () => {
  let container: HTMLElement;
  // act so that we can wait for useEffect hooks to run:
  await act(async () => {
    container = renderBoardPanel().container;
  });
  await waitFor(async () => {
    // XXX this test is actually broken. the snapshot is wrong.
    // expect(container.querySelector('.tile-p0')).not.toBeNull();
    // console.log('ok', container.querySelector('.tile-p0'));
    expect(container).toMatchSnapshot();
  });
});
