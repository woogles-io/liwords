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

export function infer(req_str: string): string;

export function infer_free(handle: number): void;

export function memory_held(): number;

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

export function sim_set(sim: number, req_str: string): string;

export function sim_step(sim: number, req_str: string): string;

export function stitched_z(iterations: number, alpha: number): number;

export function threats(req_str: string): string;

export function threats_free(threats: number): void;

export function threats_new(req_str: string): number;

export function threats_step(threats: number, req_str: string): string;

export function version(): string;

export type InitInput = RequestInfo | URL | Response | BufferSource | WebAssembly.Module;

export interface InitOutput {
    readonly memory: WebAssembly.Memory;
    readonly analyze: (a: number, b: number) => [number, number, number, number];
    readonly do_this_on_startup: () => void;
    readonly endgame_free: (a: number) => [number, number];
    readonly endgame_new: (a: number, b: number) => [number, number, number];
    readonly endgame_step: (a: number, b: number, c: number) => [number, number, number, number];
    readonly evict_klv: (a: number, b: number) => [number, number, number];
    readonly evict_kwg: (a: number, b: number) => [number, number, number];
    readonly hooks: (a: number, b: number) => [number, number, number, number];
    readonly infer: (a: number, b: number) => [number, number, number, number];
    readonly infer_free: (a: number) => [number, number];
    readonly memory_held: () => number;
    readonly play_score: (a: number, b: number) => [number, number, number, number];
    readonly precache_kblv2: (a: number, b: number, c: any) => [number, number];
    readonly precache_kbwg: (a: number, b: number, c: any) => [number, number];
    readonly precache_klv: (a: number, b: number, c: any) => [number, number];
    readonly precache_kwg: (a: number, b: number, c: any) => [number, number];
    readonly sim_add: (a: number, b: number, c: number) => [number, number, number, number];
    readonly sim_free: (a: number) => [number, number];
    readonly sim_new: (a: number, b: number) => [number, number, number];
    readonly sim_pause: (a: number, b: number, c: number) => [number, number, number, number];
    readonly sim_remove: (a: number, b: number, c: number) => [number, number, number, number];
    readonly sim_resume: (a: number, b: number, c: number) => [number, number, number, number];
    readonly sim_set: (a: number, b: number, c: number) => [number, number, number, number];
    readonly sim_step: (a: number, b: number, c: number) => [number, number, number, number];
    readonly stitched_z: (a: number, b: number) => [number, number, number];
    readonly threats: (a: number, b: number) => [number, number, number, number];
    readonly threats_free: (a: number) => [number, number];
    readonly threats_new: (a: number, b: number) => [number, number, number];
    readonly threats_step: (a: number, b: number, c: number) => [number, number, number, number];
    readonly version: () => [number, number];
    readonly __wbindgen_free: (a: number, b: number, c: number) => void;
    readonly __wbindgen_exn_store: (a: number) => void;
    readonly __externref_table_alloc: () => number;
    readonly __wbindgen_externrefs: WebAssembly.Table;
    readonly __wbindgen_malloc: (a: number, b: number) => number;
    readonly __wbindgen_realloc: (a: number, b: number, c: number, d: number) => number;
    readonly __externref_table_dealloc: (a: number) => void;
    readonly __wbindgen_start: () => void;
}

export type SyncInitInput = BufferSource | WebAssembly.Module;

/**
 * Instantiates the given `module`, which can either be bytes or
 * a precompiled `WebAssembly.Module`.
 *
 * @param {{ module: SyncInitInput }} module - Passing `SyncInitInput` directly is deprecated.
 *
 * @returns {InitOutput}
 */
export function initSync(module: { module: SyncInitInput } | SyncInitInput): InitOutput;

/**
 * If `module_or_path` is {RequestInfo} or {URL}, makes a request and
 * for everything else, calls `WebAssembly.instantiate` directly.
 *
 * @param {{ module_or_path: InitInput | Promise<InitInput> }} module_or_path - Passing `InitInput` directly is deprecated.
 *
 * @returns {Promise<InitOutput>}
 */
export default function __wbg_init (module_or_path?: { module_or_path: InitInput | Promise<InitInput> } | InitInput | Promise<InitInput>): Promise<InitOutput>;
