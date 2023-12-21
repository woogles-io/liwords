/**
 * @jest-environment jsdom
 */

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

const audio = jest.fn().mockImplementation(() => ({
  pause: mocks.Audio.pause,
  play: mocks.Audio.play,
  load: mocks.Audio.load,
  addEventListener: mocks.Audio.addEventListener,
}));

Object.defineProperty(window, 'localStorage', { value: mocks.localStorage });
Object.defineProperty(window, 'Audio', { value: audio });
Object.defineProperty(window, 'TextEncoder', { value: TextEncoder });
Object.defineProperty(window, 'TextDecoder', { value: TextDecoder });
