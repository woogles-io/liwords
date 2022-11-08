// jest-dom adds custom jest matchers for asserting on DOM nodes.
// allows you to do things like:
// expect(element).toHaveTextContent(/react/i)
// learn more: https://github.com/testing-library/jest-dom
import '@testing-library/jest-dom/extend-expect';
import { TextEncoder, TextDecoder } from 'util';

// mock localStorage

const mocks = {
  Audio: {
    pause: jest.fn(),
    play: jest.fn(),
    load: jest.fn(),
    addEventListener: jest.fn(),
  },
  localStorage: {
    getItem: jest.fn(),
    setItem: jest.fn(),
    clear: jest.fn(),
  },
};

global.localStorage = mocks.localStorage;

global.Audio = jest.fn().mockImplementation(() => ({
  pause: mocks.Audio.pause,
  play: mocks.Audio.play,
  load: mocks.Audio.load,
  addEventListener: mocks.Audio.addEventListener,
}));

global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder;
