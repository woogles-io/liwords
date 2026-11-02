/* tslint:disable */
/* eslint-disable */

export function analyze(req_str: string): string;

export function do_this_on_startup(): void;

export function endgame_free(handle: number): void;

export function endgame_new(req_str: string): number;

export function endgame_step(handle: number, req_str: string): string;

export function evict_klv(key: string): boolean;

export function evict_kwg(key: string): boolean;

export function hooks(req_str: string): string;

export function play_score(req_str: string): string;

export function precache_kblv2(key: string, value: Uint8Array): void;

export function precache_kbwg(key: string, value: Uint8Array): void;

export function precache_klv(key: string, value: Uint8Array): void;

export function precache_kwg(key: string, value: Uint8Array): void;

export function sim_add(sim: number, plays_str: string): string;

export function sim_free(sim: number): void;

export function sim_new(req_str: string): number;

export function sim_pause(sim: number, plays_str: string): string;

export function sim_remove(sim: number, plays_str: string): string;

export function sim_resume(sim: number, plays_str: string): string;

export function sim_step(sim: number, req_str: string): string;

export function version(): string;
