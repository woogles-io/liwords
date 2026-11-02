/* @ts-self-types="./wolges_wasm.d.ts" */
import * as wasm from "./wolges_wasm_bg.wasm";
import { __wbg_set_wasm } from "./wolges_wasm_bg.js";

__wbg_set_wasm(wasm);
wasm.__wbindgen_start();
export {
    analyze, do_this_on_startup, endgame_free, endgame_new, endgame_step, evict_klv, evict_kwg, hooks, play_score, precache_kblv2, precache_kbwg, precache_klv, precache_kwg, sim_add, sim_free, sim_new, sim_pause, sim_remove, sim_resume, sim_step, version
} from "./wolges_wasm_bg.js";
