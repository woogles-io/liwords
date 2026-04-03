/* @ts-self-types="./wolges_wasm.d.ts" */

import * as wasm from "./wolges_wasm_bg.wasm";
import { __wbg_set_wasm } from "./wolges_wasm_bg.js";
__wbg_set_wasm(wasm);
wasm.__wbindgen_start();
export {
    analyze, do_this_on_startup, play_score, precache_kbwg, precache_klv, precache_kwg
} from "./wolges_wasm_bg.js";
