import React, { useState } from 'react';
import { useHistory } from 'react-router-dom';
import { Button, Popconfirm } from 'antd';
import {
  DoubleLeftOutlined,
  DoubleRightOutlined,
  LeftOutlined,
  RightOutlined,
} from '@ant-design/icons';
import { useMountedState } from '../utils/mounted';
import {
  useExaminableGameContextStoreContext,
  useExamineStoreContext,
  useGameContextStoreContext,
  useResetStoreContext,
} from '../store/store';
import { Unrace } from '../utils/unrace';
import { fetchAndPrecache, getMacondo } from '../wasm/loader';

const unrace = new Unrace();

// See analyzer/analyzer.go JsonMove.
type JsonMove = {
  Action: string;
  Row: number; // int
  Column: number; // int
  Vertical: boolean;
  DisplayCoordinates: string;
  Tiles: string;
  Leave: string;
  Equity: number; // float64
  Score: number; // int
};

const filesByLexicon = [
  {
    lexicons: ['CSW19', 'NWL18'],
    cacheKey: 'data/letterdistributions/english.csv',
    path: '/wasm/english.csv',
  },
  {
    lexicons: ['CSW19', 'NWL18'],
    cacheKey: 'data/strategy/default_english/leaves.idx',
    path: '/wasm/leaves.idx',
  },
  {
    lexicons: ['CSW19', 'NWL18'],
    cacheKey: 'data/strategy/default_english/preendgame.json',
    path: '/wasm/preendgame.json',
  },
  {
    lexicons: ['CSW19'],
    cacheKey: 'data/lexica/gaddag/CSW19.gaddag',
    path: '/wasm/CSW19.gaddag',
  },
  {
    lexicons: ['NWL18'],
    cacheKey: 'data/lexica/gaddag/NWL18.gaddag',
    path: '/wasm/NWL18.gaddag',
  },
];

const ExamineGameControls = React.memo((props: { lexicon: string }) => {
  const { lexicon } = props;
  const {
    gameContext: examinableGameContext,
  } = useExaminableGameContextStoreContext();
  const {
    examinedTurn,
    handleExamineEnd,
    handleExamineFirst,
    handleExaminePrev,
    handleExamineNext,
    handleExamineLast,
  } = useExamineStoreContext();
  const { gameContext } = useGameContextStoreContext();
  const numberOfTurns = gameContext.turns.length;

  const handleExaminer = React.useCallback(() => {
    (async () => {
      const {
        board: { dim, letters },
        onturn,
        players,
      } = examinableGameContext;

      const boardObj = {
        size: dim,
        rack: players[onturn].currentRack,
        board: Array.from(new Array(dim), (_, row) =>
          letters.substr(row * dim, dim)
        ),
        lexicon,
      };
      console.log(boardObj); // for debugging

      const macondo = await getMacondo();
      await unrace.run(() =>
        Promise.all(
          filesByLexicon.map(({ lexicons, cacheKey, path }) =>
            lexicons.includes(lexicon)
              ? fetchAndPrecache(macondo, cacheKey, path)
              : null
          )
        )
      );

      const boardStr = JSON.stringify(boardObj);
      const movesStr = await macondo.analyze(boardStr);
      const movesObj = JSON.parse(movesStr) as Array<JsonMove>;

      // Just log for now.
      console.log(movesObj); // for debugging

      const suggestions = [];
      for (const move of movesObj) {
        if (move.Action === 'Play') {
          // move.Tiles may have '.', read the board to complete it.
          let tiles = '';
          let r = move.Row;
          let c = move.Column;
          let inParen = false;
          for (const t of move.Tiles) {
            if (t === '.') {
              if (!inParen) {
                tiles += '(';
                inParen = true;
              }
              tiles += letters[r * dim + c];
            } else {
              if (inParen) {
                tiles += ')';
                inParen = false;
              }
              tiles += t;
            }
            if (move.Vertical) ++r;
            else ++c;
          }
          if (inParen) tiles += ')';
          suggestions.push({
            move: `${move.DisplayCoordinates} ${tiles}`,
            score: move.Score,
            leave: move.Leave,
            valuation: move.Equity,
          });
        } else if (move.Action === 'Exchange' && move.Score === 0) {
          suggestions.push({
            move: `Exch. ${move.Tiles}`,
            score: 0,
            leave: move.Leave,
            valuation: move.Equity,
          });
        } else if (move.Action === 'Pass' && move.Score === 0) {
          suggestions.push({
            move: 'Pass',
            score: 0,
            leave: move.Leave,
            valuation: move.Equity,
          });
        } else if (move.Action === '' && move.Score === 0) {
          // This happens at end of game. Ignore.
        } else {
          throw new Error(`unhandled case`);
        }
      }
      let lastValuation = 0;
      let lastRank = 1;
      const suggestionStrs = suggestions.map(
        ({ move, score, leave, valuation }, idx) => {
          if (lastValuation !== valuation) {
            lastValuation = valuation;
            lastRank = idx + 1;
          }
          return [
            String(lastRank),
            move,
            String(score),
            leave,
            valuation.toFixed(1),
          ] as Array<string>;
        }
      );
      const suggestionMaxLengths = Array.from(new Array(5), (_, idx) =>
        suggestionStrs.reduce(
          (maxLength, suggestionStr) =>
            Math.max(maxLength, suggestionStr[idx].length),
          0
        )
      );
      const formattedSuggestions = suggestionStrs.map(
        (suggestionStr) =>
          `${suggestionStr[0].padStart(
            suggestionMaxLengths[0]
          )} ${suggestionStr[1].padEnd(
            suggestionMaxLengths[1]
          )} ${suggestionStr[2].padStart(
            suggestionMaxLengths[2]
          )} ${suggestionStr[3].padEnd(
            suggestionMaxLengths[3]
          )} ${suggestionStr[4].padStart(suggestionMaxLengths[4])}`
      );
      console.log(formattedSuggestions.join('\n'));
    })();
  }, [examinableGameContext, lexicon]);

  return (
    <div className="game-controls">
      <Button onClick={handleExaminer}>Options</Button>
      <Button
        shape="circle"
        icon={<DoubleLeftOutlined />}
        type="primary"
        onClick={handleExamineFirst}
        disabled={examinedTurn <= 0 || numberOfTurns <= 0}
      />
      <Button
        shape="circle"
        icon={<LeftOutlined />}
        type="primary"
        onClick={handleExaminePrev}
        disabled={examinedTurn <= 0 || numberOfTurns <= 0}
      />
      <Button
        shape="circle"
        icon={<RightOutlined />}
        type="primary"
        onClick={handleExamineNext}
        disabled={examinedTurn >= numberOfTurns}
      />
      <Button
        shape="circle"
        icon={<DoubleRightOutlined />}
        type="primary"
        onClick={handleExamineLast}
        disabled={examinedTurn >= numberOfTurns}
      />
      <Button onClick={handleExamineEnd}>Done</Button>
    </div>
  );
});

export type Props = {
  isExamining: boolean;
  exchangeAllowed?: boolean;
  finalPassOrChallenge?: boolean;
  myTurn?: boolean;
  observer?: boolean;
  showExchangeModal: () => void;
  onPass: () => void;
  onResign: () => void;
  onRecall: () => void;
  onChallenge: () => void;
  onCommit: () => void;
  onExamine: () => void;
  onExportGCG: () => void;
  onRematch: () => void;
  gameEndControls: boolean;
  showRematch: boolean;
  currentRack: string;
  tournamentID?: string;
  lexicon: string;
};

const GameControls = React.memo((props: Props) => {
  const [passVisible, setPassVisible] = useState(false);
  const [challengeVisible, setChallengeVisible] = useState(false);
  const [resignVisible, setResignVisible] = useState(false);

  if (props.isExamining) {
    return <ExamineGameControls lexicon={props.lexicon} />;
  }

  if (props.gameEndControls) {
    return (
      <EndGameControls
        onRematch={props.onRematch}
        onExamine={props.onExamine}
        onExportGCG={props.onExportGCG}
        showRematch={props.showRematch && !props.observer}
        tournamentID={props.tournamentID}
      />
    );
  }

  if (props.observer) {
    return (
      <div className="game-controls">
        <Button onClick={props.onExamine}>Examine</Button>
      </div>
    );
  }

  // Temporary dead code.
  if (props.observer) {
    return null;
  }

  return (
    <div className="game-controls">
      <div className="secondary-controls">
        <Popconfirm
          title="Are you sure you wish to resign?"
          onCancel={() => {
            setResignVisible(false);
          }}
          onConfirm={() => {
            props.onResign();
            setResignVisible(false);
          }}
          onVisibleChange={(visible) => {
            setResignVisible(visible);
          }}
          okText="Yes"
          cancelText="No"
          visible={resignVisible}
        >
          <Button
            danger
            onDoubleClick={() => {
              props.onResign();
              setResignVisible(false);
            }}
          >
            Ragequit
          </Button>
        </Popconfirm>

        <Popconfirm
          title="Are you sure you wish to pass?"
          onCancel={() => {
            setPassVisible(false);
          }}
          onConfirm={() => {
            props.onPass();
            setPassVisible(false);
          }}
          onVisibleChange={(visible) => {
            setPassVisible(visible);
          }}
          okText="Yes"
          cancelText="No"
          visible={passVisible}
        >
          <Button
            onDoubleClick={() => {
              props.onPass();
              setPassVisible(false);
            }}
            danger
            disabled={!props.myTurn}
            type={
              props.finalPassOrChallenge && props.myTurn ? 'primary' : 'default'
            }
          >
            Pass
            <span className="key-command">2</span>
          </Button>
        </Popconfirm>
      </div>
      <div className="secondary-controls">
        <Popconfirm
          title="Are you sure you wish to challenge?"
          onCancel={() => {
            setChallengeVisible(false);
          }}
          onConfirm={() => {
            props.onChallenge();
            setChallengeVisible(false);
          }}
          onVisibleChange={(visible) => {
            setChallengeVisible(visible);
          }}
          okText="Yes"
          cancelText="No"
          visible={challengeVisible}
        >
          <Button
            onDoubleClick={() => {
              props.onChallenge();
              setChallengeVisible(false);
            }}
            disabled={!props.myTurn}
          >
            Challenge
            <span className="key-command">3</span>
          </Button>
        </Popconfirm>
        <Button
          onClick={props.showExchangeModal}
          disabled={!(props.myTurn && props.exchangeAllowed)}
        >
          Exchange
          <span className="key-command">4</span>
        </Button>
      </div>
      <Button
        type="primary"
        className="play"
        onClick={props.onCommit}
        disabled={!props.myTurn || props.finalPassOrChallenge}
      >
        Play
      </Button>
    </div>
  );
});

type EGCProps = {
  onRematch: () => void;
  showRematch: boolean;
  onExamine: () => void;
  onExportGCG: () => void;
  tournamentID?: string;
};

const EndGameControls = (props: EGCProps) => {
  const { useState } = useMountedState();

  const [rematchDisabled, setRematchDisabled] = useState(false);
  const { resetStore } = useResetStoreContext();
  const history = useHistory();
  const handleExitToLobby = React.useCallback(() => {
    resetStore();
    props.tournamentID
      ? history.replace(`/tournament/${props.tournamentID}`)
      : history.replace('/');
  }, [history, resetStore, props.tournamentID]);

  return (
    <div className="game-controls">
      <div className="secondary-controls">
        <Button>Options</Button>
        <Button onClick={props.onExamine}>Examine</Button>
      </div>
      <div className="secondary-controls">
        <Button onClick={props.onExportGCG}>Export GCG</Button>
        <Button onClick={handleExitToLobby}>Exit</Button>
      </div>
      {props.showRematch && !rematchDisabled && (
        <Button
          type="primary"
          data-testid="rematch-button"
          className="play"
          onClick={() => {
            setRematchDisabled(true);
            if (!rematchDisabled) {
              props.onRematch();
            }
          }}
        >
          Rematch
        </Button>
      )}
    </div>
  );
};

export default GameControls;
