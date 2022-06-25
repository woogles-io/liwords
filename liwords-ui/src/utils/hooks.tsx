import React, { useCallback, useEffect, useMemo, useRef } from 'react';
import axios from 'axios';
import { toAPIUrl } from '../api/api';
import { DefineWordsResponse, GameMetadata } from '../gameroom/game_info';
import { GameEvent } from '../gen/macondo/api/proto/macondo/macondo_pb';
import { GameState } from '../store/reducers/game_reducer';
import { ChatEntityType, ChatEntityObj } from '../store/store';
import { Blank } from '../utils/cwgame/common';
import { useMountedState } from '../utils/mounted';
import { Unrace } from '../utils/unrace';

export const useFirefoxPatch = () => {
  useEffect(() => {
    // Prevent backspace unless we're in an input element. We don't want to
    // leave if we're on Firefox.

    const rx = /INPUT|SELECT|TEXTAREA/i;
    const evtHandler = (e: KeyboardEvent) => {
      const el = e.target as HTMLElement;
      if (e.which === 8) {
        if (
          !rx.test(el.tagName) ||
          (el as HTMLInputElement).disabled ||
          (el as HTMLInputElement).readOnly
        ) {
          e.preventDefault();
        }
      }
    };

    document.addEventListener('keydown', evtHandler);
    document.addEventListener('keypress', evtHandler);

    return () => {
      document.removeEventListener('keydown', evtHandler);
      document.removeEventListener('keypress', evtHandler);
    };
  }, []);
};

export const useDefinitionAndPhonyChecker = ({
  addChat,
  enableHoverDefine,
  gameContext,
  gameDone,
  gameID,
  gameInfo,
}: {
  addChat: (chat: ChatEntityObj) => void;
  enableHoverDefine: boolean;
  gameContext: GameState;
  gameDone: boolean;
  gameID?: string;
  gameInfo: GameMetadata;
}) => {
  const { useState } = useMountedState();

  // undefined = not known
  const [wordInfo, setWordInfo] = useState<{
    [key: string]: undefined | { v: boolean; d: string };
  }>({});
  const wordInfoRef = useRef(wordInfo);
  wordInfoRef.current = wordInfo;
  const [unrace, setUnrace] = useState(new Unrace());
  // undefined = not ready to report
  // null = game may have ended, check if ready to report
  const [phonies, setPhonies] = useState<undefined | null | Array<string>>(
    undefined
  );

  const [showDefinitionHover, setShowDefinitionHover] = useState<
    { x: number; y: number; words: Array<string> } | undefined
  >(undefined);
  const [willHideDefinitionHover, setWillHideDefinitionHover] = useState(false);

  const anagrams = gameInfo.game_request.rules.variant_name === 'wordsmog';
  const [definedAnagram, setDefinedAnagram] = useState(0);
  const definedAnagramRef = useRef(definedAnagram);
  definedAnagramRef.current = definedAnagram;

  const definitionPopover = useMemo(() => {
    if (!showDefinitionHover) return undefined;
    const entries = [];
    const numAnagramsEach = [];
    for (const word of showDefinitionHover.words) {
      const uppercasedWord = word.toUpperCase();
      const definition = wordInfo[uppercasedWord];
      // if phony-checker returned {v:true,d:""}, wait for definition to load
      if (definition && !(definition.v && !definition.d)) {
        if (anagrams && definition.v) {
          const shortList = []; // list of words and invalid entries
          const anagramDefinitions = []; // defined words
          for (const singleEntry of definition.d.split('\n')) {
            const m = singleEntry.match(/^([^-]*) - (.*)$/m);
            if (m) {
              const [, actualWord, actualDefinition] = m;
              anagramDefinitions.push({
                word: actualWord,
                definition: (
                  <React.Fragment>
                    <span className="defined-word">{actualWord}</span> -{' '}
                    {actualDefinition}
                  </React.Fragment>
                ),
              });
              shortList.push(actualWord);
            } else {
              shortList.push(singleEntry);
            }
          }
          const defineWhich =
            anagramDefinitions.length > 0
              ? definedAnagramRef.current % anagramDefinitions.length
              : 0;
          const anagramDefinition = anagramDefinitions[defineWhich];
          entries.push(
            <li key={entries.length} className="definition-entry">
              {uppercasedWord} -{' '}
              {shortList.map((word, idx) => (
                <React.Fragment key={idx}>
                  {idx > 0 && ', '}
                  {word === anagramDefinition?.word ? (
                    <span className="defined-word">{word}</span>
                  ) : (
                    word
                  )}
                </React.Fragment>
              ))}
            </li>
          );
          if (anagramDefinitions.length > 0) {
            numAnagramsEach.push(anagramDefinitions.length);
            entries.push(
              <li key={entries.length} className="definition-entry">
                {anagramDefinition.definition}
              </li>
            );
          }
        } else {
          entries.push(
            <li key={entries.length} className="definition-entry">
              <span className="defined-word">
                {uppercasedWord}
                {definition.v ? '' : '*'}
              </span>{' '}
              -{' '}
              {definition.v ? (
                <span className="definition">{String(definition.d)}</span>
              ) : (
                <span className="invalid-word">
                  {anagrams ? 'no valid words' : 'not a word'}
                </span>
              )}
            </li>
          );
        }
      }
    }
    if (numAnagramsEach.length > 0) {
      const numAnagramsLCM = numAnagramsEach.reduce((a, b) => {
        const ab = a * b;
        while (b !== 0) {
          const t = b;
          b = a % b;
          a = t;
        }
        return ab / a; // a = gcd, so ab/a = lcm
      });
      setDefinedAnagram((definedAnagramRef.current + 1) % numAnagramsLCM);
    } else {
      setDefinedAnagram(0);
    }
    if (!entries.length) return undefined;
    return {
      x: showDefinitionHover.x,
      y: showDefinitionHover.y,
      content: <ul className="definitions">{entries}</ul>,
    };
  }, [anagrams, showDefinitionHover, wordInfo]);

  const hideDefinitionHover = useCallback(() => {
    setShowDefinitionHover(undefined);
  }, []);

  useEffect(() => {
    if (willHideDefinitionHover) {
      // if the pointer is moved out of a tile, the definition is not hidden
      // immediately. this is an intentional design decision to improve
      // usability and responsiveness, and it enables smoother transition if
      // the pointer is moved to a nearby tile.
      const t = setTimeout(() => {
        hideDefinitionHover();
      }, 1000);
      return () => clearTimeout(t);
    }
  }, [willHideDefinitionHover, hideDefinitionHover]);

  const handleSetHover = useCallback(
    (x: number, y: number, words: Array<string> | undefined) => {
      if (enableHoverDefine && words) {
        setWillHideDefinitionHover(false);
        setShowDefinitionHover((oldValue) => {
          const newValue = {
            x,
            y,
            words,
            definedAnagram,
          };
          // if the pointer is moved out of a tile and back in, and the words
          // formed have not changed, reuse the object to avoid rerendering.
          if (JSON.stringify(oldValue) === JSON.stringify(newValue)) {
            return oldValue;
          }
          return newValue;
        });
      } else {
        setWillHideDefinitionHover(true);
      }
    },
    [enableHoverDefine, definedAnagram]
  );

  const [playedWords, setPlayedWords] = useState(new Set<string>());
  useEffect(() => {
    setPlayedWords((oldPlayedWords) => {
      const playedWords = new Set(oldPlayedWords);
      for (const turn of gameContext.turns) {
        for (const word of turn.getWordsFormedList()) {
          playedWords.add(word);
        }
      }
      return playedWords.size === oldPlayedWords.size
        ? oldPlayedWords
        : playedWords;
    });
  }, [gameContext]);

  useEffect(() => {
    // forget everything if it goes to a new game
    setWordInfo({});
    setPlayedWords(new Set());
    setUnrace(new Unrace());
    setPhonies(undefined);
    setShowDefinitionHover(undefined);
  }, [gameID, gameInfo.game_request.lexicon]);

  useEffect(() => {
    if (gameDone || showDefinitionHover) {
      // when definition is requested, get definitions for all words (up to
      // that point) that have not yet been defined. this is an intentional
      // design decision to improve usability and responsiveness.
      setWordInfo((oldWordInfo) => {
        let wordInfo = oldWordInfo;
        playedWords.forEach((word) => {
          if (!(word in wordInfo)) {
            if (wordInfo === oldWordInfo) wordInfo = { ...oldWordInfo };
            wordInfo[word] = undefined;
          }
        });
        if (showDefinitionHover) {
          // also define tentative words (mostly from examiner) if no undesignated blanks.
          for (const word of showDefinitionHover.words) {
            if (!word.includes(Blank)) {
              const uppercasedWord = word.toUpperCase();
              if (!(uppercasedWord in wordInfo)) {
                if (wordInfo === oldWordInfo) wordInfo = { ...oldWordInfo };
                wordInfo[uppercasedWord] = undefined;
              }
            }
          }
        }
        setPhonies((oldValue) => oldValue ?? null);
        return wordInfo;
      });
    }
  }, [playedWords, gameDone, showDefinitionHover]);

  useEffect(() => {
    const cancelTokenSource = axios.CancelToken.source();
    unrace.run(async () => {
      const wordInfo = wordInfoRef.current; // take the latest version after unrace
      const wordsToDefine: Array<string> = [];
      for (const word in wordInfo) {
        const definition = wordInfo[word];
        if (
          definition === undefined ||
          (showDefinitionHover && definition.v && !definition.d)
        ) {
          wordsToDefine.push(word);
        }
      }
      if (!wordsToDefine.length) return;
      wordsToDefine.sort(); // mitigate OCD
      const lexicon = gameInfo.game_request.lexicon;
      try {
        const defineResp = await axios.post<DefineWordsResponse>(
          toAPIUrl('word_service.WordService', 'DefineWords'),
          {
            lexicon,
            words: wordsToDefine,
            definitions: !!showDefinitionHover,
            anagrams,
          },
          { cancelToken: cancelTokenSource.token }
        );
        if (showDefinitionHover) {
          // for certain lexicons, try getting definitions from other sources
          for (const otherLexicon of lexicon === 'ECWL'
            ? ['CSW21', 'NWL20']
            : lexicon === 'CSW19X'
            ? ['CSW21']
            : []) {
            const wordsToRedefine = [];
            for (const word of wordsToDefine) {
              if (
                defineResp.data.results[word]?.v &&
                defineResp.data.results[word].d === word
              ) {
                wordsToRedefine.push(word);
              }
            }
            if (!wordsToRedefine.length) break;
            const otherDefineResp = await axios.post<DefineWordsResponse>(
              toAPIUrl('word_service.WordService', 'DefineWords'),
              {
                lexicon: otherLexicon,
                words: wordsToRedefine,
                definitions: !!showDefinitionHover,
                anagrams,
              },
              { cancelToken: cancelTokenSource.token }
            );
            for (const word of wordsToRedefine) {
              const newDefinition = otherDefineResp.data.results[word].d;
              if (newDefinition && newDefinition !== word) {
                defineResp.data.results[word].d = newDefinition;
              }
            }
          }
        }
        setWordInfo((oldWordInfo) => {
          const wordInfo = { ...oldWordInfo };
          for (const word of wordsToDefine) {
            wordInfo[word] = defineResp.data.results[word];
          }
          return wordInfo;
        });
      } catch (e) {
        if (axios.isCancel(e)) {
          // request canceled because it is no longer relevant.
        } else {
          // no definitions then... sadpepe.
          console.log('cannot check words', e);
        }
      }
    });
    return () => {
      cancelTokenSource.cancel();
    };
  }, [
    anagrams,
    showDefinitionHover,
    gameInfo.game_request.lexicon,
    wordInfo,
    unrace,
  ]);

  useEffect(() => {
    if (phonies === null) {
      if (gameDone) {
        const phonies: Array<string> = [];
        let hasWords = false; // avoid running this before the first GameHistoryRefresher event
        for (const word of Array.from(playedWords)) {
          hasWords = true;
          const definition = wordInfo[word];
          if (definition === undefined) {
            // not ready (this should not happen though)
            return;
          } else if (!definition.v) {
            phonies.push(word);
          }
        }
        if (hasWords) {
          phonies.sort();
          setPhonies(phonies);
          return;
        }
      }
      setPhonies(undefined); // not ready to display
    }
  }, [gameDone, phonies, playedWords, wordInfo]);

  const lastPhonyReport = useRef('');
  useEffect(() => {
    if (!phonies) return;
    if (phonies.length) {
      // since +false === 0 and +true === 1, this is [unchallenged, challenged]
      const groupedWords = [new Set(), new Set()];
      let returningTiles = false;
      for (let i = gameContext.turns.length; --i >= 0; ) {
        const turn = gameContext.turns[i];
        if (turn.getType() === GameEvent.Type.PHONY_TILES_RETURNED) {
          returningTiles = true;
        } else {
          for (const word of turn.getWordsFormedList()) {
            groupedWords[+returningTiles].add(word);
          }
          returningTiles = false;
        }
      }
      // note that a phony can appear in both lists
      const unchallengedPhonies = phonies.filter((word) =>
        groupedWords[0].has(word)
      );
      const challengedPhonies = phonies.filter((word) =>
        groupedWords[1].has(word)
      );
      const thisPhonyReport = JSON.stringify({
        challengedPhonies,
        unchallengedPhonies,
      });
      if (lastPhonyReport.current !== thisPhonyReport) {
        lastPhonyReport.current = thisPhonyReport;
        if (challengedPhonies.length) {
          addChat({
            entityType: ChatEntityType.ErrorMsg,
            sender: '',
            message: `Invalid words challenged off: ${challengedPhonies
              .map((x) => `${x}*`)
              .join(', ')}`,
            channel: 'server',
          });
        }
        if (unchallengedPhonies.length) {
          addChat({
            entityType: ChatEntityType.ErrorMsg,
            sender: '',
            message: `Invalid words played and not challenged: ${unchallengedPhonies
              .map((x) => `${x}*`)
              .join(', ')}`,
            channel: 'server',
          });
        }
      }
    } else {
      const thisPhonyReport = 'all valid';
      if (lastPhonyReport.current !== thisPhonyReport) {
        lastPhonyReport.current = thisPhonyReport;
        addChat({
          entityType: ChatEntityType.ServerMsg,
          sender: '',
          message: 'All words played are valid',
          channel: 'server',
        });
      }
    }
  }, [gameContext, phonies, addChat]);

  return {
    handleSetHover,
    hideDefinitionHover,
    definitionPopover,
  };
};
