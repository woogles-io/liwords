
var MAGPIE = (() => {
  var _scriptDir = import.meta.url;
  
  return (
function(moduleArg = {}) {

var f = moduleArg, aa, ba;
f.ready = new Promise((a, b) => {
  aa = a;
  ba = b;
});
"_free _main _malloc _precache_file_data _process_ucgi_command_wasm _score_play _destroy_cache _ucgi_search_status_wasm _ucgi_stop_search_wasm _static_evaluation __emscripten_thread_init __emscripten_thread_exit __emscripten_thread_crashed __emscripten_thread_mailbox_await __emscripten_tls_init _pthread_self checkMailbox establishStackSpace invokeEntryPoint PThread _fflush __emscripten_check_mailbox onRuntimeInitialized".split(" ").forEach(a => {
  Object.getOwnPropertyDescriptor(f.ready, a) || Object.defineProperty(f.ready, a, {get:() => l("You are getting " + a + " on the Promise object, instead of the instance. Use .then() to get called back with the instance, see the MODULARIZE docs in src/settings.js"), set:() => l("You are setting " + a + " on the Promise object, instead of the instance. Use .then() to get called back with the instance, see the MODULARIZE docs in src/settings.js"),});
});
var ca = Object.assign({}, f), da = (a, b) => {
  throw b;
}, ea = "object" == typeof window, m = "function" == typeof importScripts, fa = "object" == typeof process && "object" == typeof process.Za && "string" == typeof process.Za.node, ha = !ea && !fa && !m;
if (f.ENVIRONMENT) {
  throw Error("Module.ENVIRONMENT has been deprecated. To force the environment, use the ENVIRONMENT compile-time option (for example, -sENVIRONMENT=web or -sENVIRONMENT=node)");
}
var p = f.ENVIRONMENT_IS_PTHREAD || !1, q = "";
function ia(a) {
  return f.locateFile ? f.locateFile(a, q) : q + a;
}
var ja;
if (ha) {
  if ("object" == typeof process && "function" === typeof require || "object" == typeof window || "function" == typeof importScripts) {
    throw Error("not compiled for this environment (did you build to HTML and try to run it not on the web, or set ENVIRONMENT to something - like node - and run it someplace else - like on the web?)");
  }
  ja = a => {
    if ("function" == typeof readbuffer) {
      return new Uint8Array(readbuffer(a));
    }
    a = read(a, "binary");
    r("object" == typeof a);
    return a;
  };
  "undefined" == typeof clearTimeout && (globalThis.clearTimeout = () => {
  });
  "undefined" == typeof setTimeout && (globalThis.setTimeout = a => "function" == typeof a ? a() : l());
  "function" == typeof quit && (da = (a, b) => {
    setTimeout(() => {
      if (!(b instanceof ka)) {
        let c = b;
        b && "object" == typeof b && b.stack && (c = [b, b.stack]);
        v(`exiting due to exception: ${c}`);
      }
      quit(a);
    });
    throw b;
  });
  "undefined" != typeof print && ("undefined" == typeof console && (console = {}), console.log = print, console.warn = console.error = "undefined" != typeof printErr ? printErr : print);
} else if (ea || m) {
  m ? q = self.location.href : "undefined" != typeof document && document.currentScript && (q = document.currentScript.src);
  _scriptDir && (q = _scriptDir);
  0 !== q.indexOf("blob:") ? q = q.substr(0, q.replace(/[?#].*/, "").lastIndexOf("/") + 1) : q = "";
  if ("object" != typeof window && "function" != typeof importScripts) {
    throw Error("not compiled for this environment (did you build to HTML and try to run it not on the web, or set ENVIRONMENT to something - like node - and run it someplace else - like on the web?)");
  }
  m && (ja = a => {
    var b = new XMLHttpRequest();
    b.open("GET", a, !1);
    b.responseType = "arraybuffer";
    b.send(null);
    return new Uint8Array(b.response);
  });
} else {
  throw Error("environment detection error");
}
var la = f.print || console.log.bind(console), v = f.printErr || console.error.bind(console);
Object.assign(f, ca);
ca = null;
Object.getOwnPropertyDescriptor(f, "fetchSettings") && l("`Module.fetchSettings` was supplied but `fetchSettings` not included in INCOMING_MODULE_JS_API");
w("arguments", "arguments_");
w("thisProgram", "thisProgram");
f.quit && (da = f.quit);
w("quit", "quit_");
r("undefined" == typeof f.memoryInitializerPrefixURL, "Module.memoryInitializerPrefixURL option was removed, use Module.locateFile instead");
r("undefined" == typeof f.pthreadMainPrefixURL, "Module.pthreadMainPrefixURL option was removed, use Module.locateFile instead");
r("undefined" == typeof f.cdInitializerPrefixURL, "Module.cdInitializerPrefixURL option was removed, use Module.locateFile instead");
r("undefined" == typeof f.filePackagePrefixURL, "Module.filePackagePrefixURL option was removed, use Module.locateFile instead");
r("undefined" == typeof f.read, "Module.read option was removed (modify read_ in JS)");
r("undefined" == typeof f.readAsync, "Module.readAsync option was removed (modify readAsync in JS)");
r("undefined" == typeof f.readBinary, "Module.readBinary option was removed (modify readBinary in JS)");
r("undefined" == typeof f.setWindowTitle, "Module.setWindowTitle option was removed (modify setWindowTitle in JS)");
r("undefined" == typeof f.TOTAL_MEMORY, "Module.TOTAL_MEMORY has been renamed Module.INITIAL_MEMORY");
w("read", "read_");
w("readAsync", "readAsync");
w("readBinary", "readBinary");
w("setWindowTitle", "setWindowTitle");
r(ea || m || fa, "Pthreads do not work in this environment yet (need Web Workers, or an alternative to them)");
r(!fa, "node environment detected but not enabled at build time.  Add 'node' to `-sENVIRONMENT` to enable.");
r(!ha, "shell environment detected but not enabled at build time.  Add 'shell' to `-sENVIRONMENT` to enable.");
var ma;
f.wasmBinary && (ma = f.wasmBinary);
w("wasmBinary", "wasmBinary");
var noExitRuntime = f.noExitRuntime || !0;
w("noExitRuntime", "noExitRuntime");
"object" != typeof WebAssembly && l("no native wasm support detected");
var z, na, A = !1, C;
function r(a, b) {
  a || l("Assertion failed" + (b ? ": " + b : ""));
}
var G, oa, pa, H, I, qa;
r(!f.STACK_SIZE, "STACK_SIZE can no longer be set at runtime.  Use -sSTACK_SIZE at link time");
r("undefined" != typeof Int32Array && "undefined" !== typeof Float64Array && void 0 != Int32Array.prototype.subarray && void 0 != Int32Array.prototype.set, "JS engine does not provide full typed array support");
var ra = f.INITIAL_MEMORY || 134217728;
w("INITIAL_MEMORY", "INITIAL_MEMORY");
r(65536 <= ra, "INITIAL_MEMORY should be larger than STACK_SIZE, was " + ra + "! (STACK_SIZE=65536)");
if (p) {
  z = f.wasmMemory;
} else {
  if (f.wasmMemory) {
    z = f.wasmMemory;
  } else {
    if (z = new WebAssembly.Memory({initial:ra / 65536, maximum:ra / 65536, shared:!0}), !(z.buffer instanceof SharedArrayBuffer)) {
      throw v("requested a shared WebAssembly.Memory but the returned buffer is not a SharedArrayBuffer, indicating that while the browser has SharedArrayBuffer it does not have WebAssembly threads support - you may need to set a flag"), fa && v("(on node you may need: --experimental-wasm-threads --experimental-wasm-bulk-memory and/or recent version)"), Error("bad memory");
    }
  }
}
var J = z.buffer;
f.HEAP8 = G = new Int8Array(J);
f.HEAP16 = pa = new Int16Array(J);
f.HEAP32 = H = new Int32Array(J);
f.HEAPU8 = oa = new Uint8Array(J);
f.HEAPU16 = new Uint16Array(J);
f.HEAPU32 = I = new Uint32Array(J);
f.HEAPF32 = new Float32Array(J);
f.HEAPF64 = qa = new Float64Array(J);
ra = z.buffer.byteLength;
r(0 === ra % 65536);
var sa;
function ta() {
  var a = ua();
  r(0 == (a & 3));
  0 == a && (a += 4);
  I[a >> 2] = 34821223;
  I[a + 4 >> 2] = 2310721022;
  I[0] = 1668509029;
}
function va() {
  if (!A) {
    var a = ua();
    0 == a && (a += 4);
    var b = I[a >> 2], c = I[a + 4 >> 2];
    34821223 == b && 2310721022 == c || l(`Stack overflow! Stack cookie has been overwritten at ${wa(a)}, expected hex dwords 0x89BACDFE and 0x2135467, but received ${wa(c)} ${wa(b)}`);
    1668509029 != I[0] && l("Runtime error: The application has corrupted its heap memory area (address zero)!");
  }
}
var xa = new Int16Array(1), ya = new Int8Array(xa.buffer);
xa[0] = 25459;
if (115 !== ya[0] || 99 !== ya[1]) {
  throw "Runtime error: expected the system to be little-endian! (Run with -sSUPPORT_BIG_ENDIAN to bypass)";
}
var za = [], Aa = [], Ba = [], Ca = [], Da = !1, Ea = 0;
function Fa() {
  return noExitRuntime || 0 < Ea;
}
function Ga() {
  r(!Da);
  Da = !0;
  if (!p) {
    va();
    if (!f.noFSInit && !Ha) {
      r(!Ha, "FS.init was previously called. If you want to initialize later with custom parameters, remove any earlier calls (note that one is automatically added to the generated code)");
      Ha = !0;
      Ia();
      f.stdin = f.stdin;
      f.stdout = f.stdout;
      f.stderr = f.stderr;
      f.stdin ? Ja("stdin", f.stdin) : Ka("/dev/tty", "/dev/stdin");
      f.stdout ? Ja("stdout", null, f.stdout) : Ka("/dev/tty", "/dev/stdout");
      f.stderr ? Ja("stderr", null, f.stderr) : Ka("/dev/tty1", "/dev/stderr");
      var a = La("/dev/stdin", 0), b = La("/dev/stdout", 1), c = La("/dev/stderr", 1);
      r(0 === a.o, `invalid handle for stdin (${a.o})`);
      r(1 === b.o, `invalid handle for stdout (${b.o})`);
      r(2 === c.o, `invalid handle for stderr (${c.o})`);
    }
    Ma = !1;
    Na(Aa);
  }
}
r(Math.imul, "This browser does not support Math.imul(), build with LEGACY_VM_SUPPORT or POLYFILL_OLD_MATH_FUNCTIONS to add in a polyfill");
r(Math.fround, "This browser does not support Math.fround(), build with LEGACY_VM_SUPPORT or POLYFILL_OLD_MATH_FUNCTIONS to add in a polyfill");
r(Math.clz32, "This browser does not support Math.clz32(), build with LEGACY_VM_SUPPORT or POLYFILL_OLD_MATH_FUNCTIONS to add in a polyfill");
r(Math.trunc, "This browser does not support Math.trunc(), build with LEGACY_VM_SUPPORT or POLYFILL_OLD_MATH_FUNCTIONS to add in a polyfill");
var K = 0, L = null, Oa = null, Pa = {};
function Qa(a) {
  K++;
  f.monitorRunDependencies && f.monitorRunDependencies(K);
  a ? (r(!Pa[a]), Pa[a] = 1, null === L && "undefined" != typeof setInterval && (L = setInterval(() => {
    if (A) {
      clearInterval(L), L = null;
    } else {
      var b = !1, c;
      for (c in Pa) {
        b || (b = !0, v("still waiting on run dependencies:")), v("dependency: " + c);
      }
      b && v("(end of list)");
    }
  }, 10000))) : v("warning: run dependency added without ID");
}
function Ra(a) {
  K--;
  f.monitorRunDependencies && f.monitorRunDependencies(K);
  a ? (r(Pa[a]), delete Pa[a]) : v("warning: run dependency removed without ID");
  0 == K && (null !== L && (clearInterval(L), L = null), Oa && (a = Oa, Oa = null, a()));
}
function l(a) {
  if (f.onAbort) {
    f.onAbort(a);
  }
  a = "Aborted(" + a + ")";
  v(a);
  A = !0;
  C = 1;
  a = new WebAssembly.RuntimeError(a);
  ba(a);
  throw a;
}
function Sa(a) {
  return a.startsWith("data:application/octet-stream;base64,");
}
function N(a) {
  return function() {
    var b = f.asm;
    r(Da, "native function `" + a + "` called before runtime initialization");
    b[a] || r(b[a], "exported native function `" + a + "` not found");
    return b[a].apply(null, arguments);
  };
}
var O;
f.locateFile ? (O = "magpie_wasm.wasm", Sa(O) || (O = ia(O))) : O = (new URL("magpie_wasm.wasm", import.meta.url)).href;
function Ta(a) {
  try {
    if (a == O && ma) {
      return new Uint8Array(ma);
    }
    if (ja) {
      return ja(a);
    }
    throw "both async and sync fetching of the wasm failed";
  } catch (b) {
    l(b);
  }
}
function Ua(a) {
  return ma || !ea && !m || "function" != typeof fetch ? Promise.resolve().then(() => Ta(a)) : fetch(a, {credentials:"same-origin"}).then(b => {
    if (!b.ok) {
      throw "failed to load wasm binary file at '" + a + "'";
    }
    return b.arrayBuffer();
  }).catch(() => Ta(a));
}
function Va(a, b, c) {
  return Ua(a).then(d => WebAssembly.instantiate(d, b)).then(d => d).then(c, d => {
    v("failed to asynchronously prepare wasm: " + d);
    O.startsWith("file://") && v("warning: Loading from a file URI (" + O + ") is not supported in most browsers. See https://emscripten.org/docs/getting_started/FAQ.html#how-do-i-run-a-local-webserver-for-testing-why-does-my-program-stall-in-downloading-or-preparing");
    l(d);
  });
}
function Wa(a, b) {
  var c = O;
  return ma || "function" != typeof WebAssembly.instantiateStreaming || Sa(c) || "function" != typeof fetch ? Va(c, a, b) : fetch(c, {credentials:"same-origin"}).then(d => WebAssembly.instantiateStreaming(d, a).then(b, function(e) {
    v("wasm streaming compile failed: " + e);
    v("falling back to ArrayBuffer instantiation");
    return Va(c, a, b);
  }));
}
var Xa, Ya;
function w(a, b) {
  Object.getOwnPropertyDescriptor(f, a) || Object.defineProperty(f, a, {configurable:!0, get:function() {
    l("Module." + a + " has been replaced with plain " + b + " (the initial value can be provided on Module, but after startup the value is only looked for on a local variable of that name)");
  }});
}
function Za(a) {
  return "FS_createPath" === a || "FS_createDataFile" === a || "FS_createPreloadedFile" === a || "FS_unlink" === a || "addRunDependency" === a || "FS_createLazyFile" === a || "FS_createDevice" === a || "removeRunDependency" === a;
}
(function(a, b) {
  "undefined" !== typeof globalThis && Object.defineProperty(globalThis, a, {configurable:!0, get:function() {
    P("`" + a + "` is not longer defined by emscripten. " + b);
  }});
})("buffer", "Please use HEAP8.buffer or wasmMemory.buffer");
function $a(a) {
  Object.getOwnPropertyDescriptor(f, a) || Object.defineProperty(f, a, {configurable:!0, get:function() {
    var b = "'" + a + "' was not exported. add it to EXPORTED_RUNTIME_METHODS (see the FAQ)";
    Za(a) && (b += ". Alternatively, forcing filesystem support (-sFORCE_FILESYSTEM) can export this for you");
    l(b);
  }});
}
function ka(a) {
  this.name = "ExitStatus";
  this.message = `Program terminated with exit(${a})`;
  this.status = a;
}
function ab(a) {
  a.terminate();
  a.onmessage = b => {
    v('received "' + b.data.cmd + '" command from terminated worker: ' + a.sa);
  };
}
function bb(a) {
  r(!p, "Internal Error! cleanupThread() can only ever be called from main application thread!");
  r(a, "Internal Error! Null pthread_ptr in cleanupThread!");
  a = Q.D[a];
  r(a);
  Q.Pa(a);
}
function cb(a) {
  r(!p, "Internal Error! spawnThread() can only ever be called from main application thread!");
  r(a.s, "Internal error, no pthread ptr!");
  var b = Q.xa();
  if (!b) {
    return 6;
  }
  r(!b.s, "Internal error!");
  Q.J.push(b);
  Q.D[a.s] = b;
  b.s = a.s;
  b.postMessage({cmd:"run", start_routine:a.Ra, arg:a.ta, pthread_ptr:a.s,}, a.Xa);
  return 0;
}
var db = (a, b) => {
  for (var c = 0, d = a.length - 1; 0 <= d; d--) {
    var e = a[d];
    "." === e ? a.splice(d, 1) : ".." === e ? (a.splice(d, 1), c++) : c && (a.splice(d, 1), c--);
  }
  if (b) {
    for (; c; c--) {
      a.unshift("..");
    }
  }
  return a;
}, eb = a => {
  var b = "/" === a.charAt(0), c = "/" === a.substr(-1);
  (a = db(a.split("/").filter(d => !!d), !b).join("/")) || b || (a = ".");
  a && c && (a += "/");
  return (b ? "/" : "") + a;
}, fb = a => {
  var b = /^(\/?|)([\s\S]*?)((?:\.{1,2}|[^\/]+?|)(\.[^.\/]*|))(?:[\/]*)$/.exec(a).slice(1);
  a = b[0];
  b = b[1];
  if (!a && !b) {
    return ".";
  }
  b && (b = b.substr(0, b.length - 1));
  return a + b;
}, gb = a => {
  if ("/" === a) {
    return "/";
  }
  a = eb(a);
  a = a.replace(/\/$/, "");
  var b = a.lastIndexOf("/");
  return -1 === b ? a : a.substr(b + 1);
}, hb = () => {
  if ("object" == typeof crypto && "function" == typeof crypto.getRandomValues) {
    return a => (a.set(crypto.getRandomValues(new Uint8Array(a.byteLength))), a);
  }
  l("no cryptographic support found for randomDevice. consider polyfilling it if you want to use something insecure like Math.random(), e.g. put this in a --pre-js: var crypto = { getRandomValues: (array) => { for (var i = 0; i < array.length; i++) array[i] = (Math.random()*256)|0 } };");
}, ib = a => (ib = hb())(a);
function jb() {
  for (var a = "", b = !1, c = arguments.length - 1; -1 <= c && !b; c--) {
    b = 0 <= c ? arguments[c] : "/";
    if ("string" != typeof b) {
      throw new TypeError("Arguments to path.resolve must be strings");
    }
    if (!b) {
      return "";
    }
    a = b + "/" + a;
    b = "/" === b.charAt(0);
  }
  a = db(a.split("/").filter(d => !!d), !b).join("/");
  return (b ? "/" : "") + a || ".";
}
var kb = a => {
  for (var b = 0, c = 0; c < a.length; ++c) {
    var d = a.charCodeAt(c);
    127 >= d ? b++ : 2047 >= d ? b += 2 : 55296 <= d && 57343 >= d ? (b += 4, ++c) : b += 3;
  }
  return b;
}, lb = (a, b, c, d) => {
  r("string" === typeof a);
  if (!(0 < d)) {
    return 0;
  }
  var e = c;
  d = c + d - 1;
  for (var g = 0; g < a.length; ++g) {
    var k = a.charCodeAt(g);
    if (55296 <= k && 57343 >= k) {
      var t = a.charCodeAt(++g);
      k = 65536 + ((k & 1023) << 10) | t & 1023;
    }
    if (127 >= k) {
      if (c >= d) {
        break;
      }
      b[c++] = k;
    } else {
      if (2047 >= k) {
        if (c + 1 >= d) {
          break;
        }
        b[c++] = 192 | k >> 6;
      } else {
        if (65535 >= k) {
          if (c + 2 >= d) {
            break;
          }
          b[c++] = 224 | k >> 12;
        } else {
          if (c + 3 >= d) {
            break;
          }
          1114111 < k && P("Invalid Unicode code point " + wa(k) + " encountered when serializing a JS string to a UTF-8 string in wasm memory! (Valid unicode code points should be in range 0-0x10FFFF).");
          b[c++] = 240 | k >> 18;
          b[c++] = 128 | k >> 12 & 63;
        }
        b[c++] = 128 | k >> 6 & 63;
      }
      b[c++] = 128 | k & 63;
    }
  }
  b[c] = 0;
  return c - e;
};
function mb(a, b) {
  var c = Array(kb(a) + 1);
  a = lb(a, c, 0, c.length);
  b && (c.length = a);
  return c;
}
var nb = "undefined" != typeof TextDecoder ? new TextDecoder("utf8") : void 0, ob = (a, b, c) => {
  var d = b + c;
  for (c = b; a[c] && !(c >= d);) {
    ++c;
  }
  if (16 < c - b && a.buffer && nb) {
    return nb.decode(a.buffer instanceof SharedArrayBuffer ? a.slice(b, c) : a.subarray(b, c));
  }
  for (d = ""; b < c;) {
    var e = a[b++];
    if (e & 128) {
      var g = a[b++] & 63;
      if (192 == (e & 224)) {
        d += String.fromCharCode((e & 31) << 6 | g);
      } else {
        var k = a[b++] & 63;
        224 == (e & 240) ? e = (e & 15) << 12 | g << 6 | k : (240 != (e & 248) && P("Invalid UTF-8 leading byte " + wa(e) + " encountered when deserializing a UTF-8 string in wasm memory to a JS string!"), e = (e & 7) << 18 | g << 12 | k << 6 | a[b++] & 63);
        65536 > e ? d += String.fromCharCode(e) : (e -= 65536, d += String.fromCharCode(55296 | e >> 10, 56320 | e & 1023));
      }
    } else {
      d += String.fromCharCode(e);
    }
  }
  return d;
}, pb = [];
function qb(a, b) {
  pb[a] = {input:[], m:[], C:b};
  rb(a, sb);
}
var sb = {open:function(a) {
  var b = pb[a.node.N];
  if (!b) {
    throw new R(43);
  }
  a.j = b;
  a.seekable = !1;
}, close:function(a) {
  a.j.C.R(a.j);
}, R:function(a) {
  a.j.C.R(a.j);
}, read:function(a, b, c, d) {
  if (!a.j || !a.j.C.ga) {
    throw new R(60);
  }
  for (var e = 0, g = 0; g < d; g++) {
    try {
      var k = a.j.C.ga(a.j);
    } catch (t) {
      throw new R(29);
    }
    if (void 0 === k && 0 === e) {
      throw new R(6);
    }
    if (null === k || void 0 === k) {
      break;
    }
    e++;
    b[c + g] = k;
  }
  e && (a.node.timestamp = Date.now());
  return e;
}, write:function(a, b, c, d) {
  if (!a.j || !a.j.C.Z) {
    throw new R(60);
  }
  try {
    for (var e = 0; e < d; e++) {
      a.j.C.Z(a.j, b[c + e]);
    }
  } catch (g) {
    throw new R(29);
  }
  d && (a.node.timestamp = Date.now());
  return e;
}}, tb = {ga:function(a) {
  if (!a.input.length) {
    var b = null;
    "undefined" != typeof window && "function" == typeof window.prompt ? (b = window.prompt("Input: "), null !== b && (b += "\n")) : "function" == typeof readline && (b = readline(), null !== b && (b += "\n"));
    if (!b) {
      return null;
    }
    a.input = mb(b, !0);
  }
  return a.input.shift();
}, Z:function(a, b) {
  null === b || 10 === b ? (la(ob(a.m, 0)), a.m = []) : 0 != b && a.m.push(b);
}, R:function(a) {
  a.m && 0 < a.m.length && (la(ob(a.m, 0)), a.m = []);
}, Ca:function() {
  return {gb:25856, ib:5, fb:191, hb:35387, eb:[3, 28, 127, 21, 4, 0, 1, 0, 17, 19, 26, 0, 18, 15, 23, 22, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,]};
}, Da:function() {
  return 0;
}, Ea:function() {
  return [24, 80];
}}, ub = {Z:function(a, b) {
  null === b || 10 === b ? (v(ob(a.m, 0)), a.m = []) : 0 != b && a.m.push(b);
}, R:function(a) {
  a.m && 0 < a.m.length && (v(ob(a.m, 0)), a.m = []);
}}, S = {u:null, B:function() {
  return S.createNode(null, "/", 16895, 0);
}, createNode:function(a, b, c, d) {
  if (24576 === (c & 61440) || 4096 === (c & 61440)) {
    throw new R(63);
  }
  S.u || (S.u = {dir:{node:{G:S.h.G, v:S.h.v, M:S.h.M, T:S.h.T, ma:S.h.ma, ra:S.h.ra, na:S.h.na, la:S.h.la, V:S.h.V}, stream:{I:S.i.I}}, file:{node:{G:S.h.G, v:S.h.v}, stream:{I:S.i.I, read:S.i.read, write:S.i.write, ba:S.i.ba, ia:S.i.ia, ka:S.i.ka}}, link:{node:{G:S.h.G, v:S.h.v, O:S.h.O}, stream:{}}, da:{node:{G:S.h.G, v:S.h.v}, stream:vb}});
  c = wb(a, b, c, d);
  16384 === (c.mode & 61440) ? (c.h = S.u.dir.node, c.i = S.u.dir.stream, c.g = {}) : 32768 === (c.mode & 61440) ? (c.h = S.u.file.node, c.i = S.u.file.stream, c.l = 0, c.g = null) : 40960 === (c.mode & 61440) ? (c.h = S.u.link.node, c.i = S.u.link.stream) : 8192 === (c.mode & 61440) && (c.h = S.u.da.node, c.i = S.u.da.stream);
  c.timestamp = Date.now();
  a && (a.g[b] = c, a.timestamp = c.timestamp);
  return c;
}, nb:function(a) {
  return a.g ? a.g.subarray ? a.g.subarray(0, a.l) : new Uint8Array(a.g) : new Uint8Array(0);
}, ea:function(a, b) {
  var c = a.g ? a.g.length : 0;
  c >= b || (b = Math.max(b, c * (1048576 > c ? 2.0 : 1.125) >>> 0), 0 != c && (b = Math.max(b, 256)), c = a.g, a.g = new Uint8Array(b), 0 < a.l && a.g.set(c.subarray(0, a.l), 0));
}, Oa:function(a, b) {
  if (a.l != b) {
    if (0 == b) {
      a.g = null, a.l = 0;
    } else {
      var c = a.g;
      a.g = new Uint8Array(b);
      c && a.g.set(c.subarray(0, Math.min(b, a.l)));
      a.l = b;
    }
  }
}, h:{G:function(a) {
  var b = {};
  b.mb = 8192 === (a.mode & 61440) ? a.id : 1;
  b.pb = a.id;
  b.mode = a.mode;
  b.rb = 1;
  b.uid = 0;
  b.ob = 0;
  b.N = a.N;
  16384 === (a.mode & 61440) ? b.size = 4096 : 32768 === (a.mode & 61440) ? b.size = a.l : 40960 === (a.mode & 61440) ? b.size = a.link.length : b.size = 0;
  b.bb = new Date(a.timestamp);
  b.qb = new Date(a.timestamp);
  b.kb = new Date(a.timestamp);
  b.ua = 4096;
  b.cb = Math.ceil(b.size / b.ua);
  return b;
}, v:function(a, b) {
  void 0 !== b.mode && (a.mode = b.mode);
  void 0 !== b.timestamp && (a.timestamp = b.timestamp);
  void 0 !== b.size && S.Oa(a, b.size);
}, M:function() {
  throw xb[44];
}, T:function(a, b, c, d) {
  return S.createNode(a, b, c, d);
}, ma:function(a, b, c) {
  if (16384 === (a.mode & 61440)) {
    try {
      var d = yb(b, c);
    } catch (g) {
    }
    if (d) {
      for (var e in d.g) {
        throw new R(55);
      }
    }
  }
  delete a.parent.g[a.name];
  a.parent.timestamp = Date.now();
  a.name = c;
  b.g[c] = a;
  b.timestamp = a.parent.timestamp;
  a.parent = b;
}, ra:function(a, b) {
  delete a.g[b];
  a.timestamp = Date.now();
}, na:function(a, b) {
  var c = yb(a, b), d;
  for (d in c.g) {
    throw new R(55);
  }
  delete a.g[b];
  a.timestamp = Date.now();
}, la:function(a) {
  var b = [".", ".."], c;
  for (c in a.g) {
    a.g.hasOwnProperty(c) && b.push(c);
  }
  return b;
}, V:function(a, b, c) {
  a = S.createNode(a, b, 41471, 0);
  a.link = c;
  return a;
}, O:function(a) {
  if (40960 !== (a.mode & 61440)) {
    throw new R(28);
  }
  return a.link;
}}, i:{read:function(a, b, c, d, e) {
  var g = a.node.g;
  if (e >= a.node.l) {
    return 0;
  }
  a = Math.min(a.node.l - e, d);
  r(0 <= a);
  if (8 < a && g.subarray) {
    b.set(g.subarray(e, e + a), c);
  } else {
    for (d = 0; d < a; d++) {
      b[c + d] = g[e + d];
    }
  }
  return a;
}, write:function(a, b, c, d, e, g) {
  r(!(b instanceof ArrayBuffer));
  if (!d) {
    return 0;
  }
  a = a.node;
  a.timestamp = Date.now();
  if (b.subarray && (!a.g || a.g.subarray)) {
    if (g) {
      return r(0 === e, "canOwn must imply no weird position inside the file"), a.g = b.subarray(c, c + d), a.l = d;
    }
    if (0 === a.l && 0 === e) {
      return a.g = b.slice(c, c + d), a.l = d;
    }
    if (e + d <= a.l) {
      return a.g.set(b.subarray(c, c + d), e), d;
    }
  }
  S.ea(a, e + d);
  if (a.g.subarray && b.subarray) {
    a.g.set(b.subarray(c, c + d), e);
  } else {
    for (g = 0; g < d; g++) {
      a.g[e + g] = b[c + g];
    }
  }
  a.l = Math.max(a.l, e + d);
  return d;
}, I:function(a, b, c) {
  1 === c ? b += a.position : 2 === c && 32768 === (a.node.mode & 61440) && (b += a.node.l);
  if (0 > b) {
    throw new R(28);
  }
  return b;
}, ba:function(a, b, c) {
  S.ea(a.node, b + c);
  a.node.l = Math.max(a.node.l, b + c);
}, ia:function(a, b, c, d, e) {
  if (32768 !== (a.node.mode & 61440)) {
    throw new R(43);
  }
  a = a.node.g;
  if (e & 2 || a.buffer !== G.buffer) {
    if (0 < c || c + b < a.length) {
      a.subarray ? a = a.subarray(c, c + b) : a = Array.prototype.slice.call(a, c, c + b);
    }
    c = !0;
    l("internal error: mmapAlloc called but `emscripten_builtin_memalign` native symbol not exported");
    b = void 0;
    if (!b) {
      throw new R(48);
    }
    G.set(a, b);
  } else {
    c = !1, b = a.byteOffset;
  }
  return {tb:b, ab:c};
}, ka:function(a, b, c, d) {
  S.i.write(a, b, 0, d, c, !1);
  return 0;
}}};
function zb(a, b) {
  var c = 0;
  a && (c |= 365);
  b && (c |= 146);
  return c;
}
var Ab = {0:"Success", 1:"Arg list too long", 2:"Permission denied", 3:"Address already in use", 4:"Address not available", 5:"Address family not supported by protocol family", 6:"No more processes", 7:"Socket already connected", 8:"Bad file number", 9:"Trying to read unreadable message", 10:"Mount device busy", 11:"Operation canceled", 12:"No children", 13:"Connection aborted", 14:"Connection refused", 15:"Connection reset by peer", 16:"File locking deadlock error", 17:"Destination address required", 
18:"Math arg out of domain of func", 19:"Quota exceeded", 20:"File exists", 21:"Bad address", 22:"File too large", 23:"Host is unreachable", 24:"Identifier removed", 25:"Illegal byte sequence", 26:"Connection already in progress", 27:"Interrupted system call", 28:"Invalid argument", 29:"I/O error", 30:"Socket is already connected", 31:"Is a directory", 32:"Too many symbolic links", 33:"Too many open files", 34:"Too many links", 35:"Message too long", 36:"Multihop attempted", 37:"File or path name too long", 
38:"Network interface is not configured", 39:"Connection reset by network", 40:"Network is unreachable", 41:"Too many open files in system", 42:"No buffer space available", 43:"No such device", 44:"No such file or directory", 45:"Exec format error", 46:"No record locks available", 47:"The link has been severed", 48:"Not enough core", 49:"No message of desired type", 50:"Protocol not available", 51:"No space left on device", 52:"Function not implemented", 53:"Socket is not connected", 54:"Not a directory", 
55:"Directory not empty", 56:"State not recoverable", 57:"Socket operation on non-socket", 59:"Not a typewriter", 60:"No such device or address", 61:"Value too large for defined data type", 62:"Previous owner died", 63:"Not super-user", 64:"Broken pipe", 65:"Protocol error", 66:"Unknown protocol", 67:"Protocol wrong type for socket", 68:"Math result not representable", 69:"Read only file system", 70:"Illegal seek", 71:"No such process", 72:"Stale file handle", 73:"Connection timed out", 74:"Text file busy", 
75:"Cross-device link", 100:"Device not a stream", 101:"Bad font file fmt", 102:"Invalid slot", 103:"Invalid request code", 104:"No anode", 105:"Block device required", 106:"Channel number out of range", 107:"Level 3 halted", 108:"Level 3 reset", 109:"Link number out of range", 110:"Protocol driver not attached", 111:"No CSI structure available", 112:"Level 2 halted", 113:"Invalid exchange", 114:"Invalid request descriptor", 115:"Exchange full", 116:"No data (for no delay io)", 117:"Timer expired", 
118:"Out of streams resources", 119:"Machine is not on the network", 120:"Package not installed", 121:"The object is remote", 122:"Advertise error", 123:"Srmount error", 124:"Communication error on send", 125:"Cross mount point (not really error)", 126:"Given log. name not unique", 127:"f.d. invalid for this operation", 128:"Remote address changed", 129:"Can   access a needed shared lib", 130:"Accessing a corrupted shared lib", 131:".lib section in a.out corrupted", 132:"Attempting to link in too many libs", 
133:"Attempting to exec a shared library", 135:"Streams pipe error", 136:"Too many users", 137:"Socket type not supported", 138:"Not supported", 139:"Protocol family not supported", 140:"Can't send after socket shutdown", 141:"Too many references", 142:"Host is down", 148:"No medium (in tape drive)", 156:"Level 2 not synchronized"}, Bb = {};
function Cb(a) {
  return a.replace(/\b_Z[\w\d_]+/g, function(b) {
    P("warning: build with -sDEMANGLE_SUPPORT to link in libcxxabi demangling");
    return b === b ? b : b + " [" + b + "]";
  });
}
var Db = null, Eb = {}, Fb = [], Gb = 1, Hb = null, Ma = !0, R = null, xb = {}, T = (a, b = {}) => {
  a = jb(a);
  if (!a) {
    return {path:"", node:null};
  }
  b = Object.assign({fa:!0, $:0}, b);
  if (8 < b.$) {
    throw new R(32);
  }
  a = a.split("/").filter(k => !!k);
  for (var c = Db, d = "/", e = 0; e < a.length; e++) {
    var g = e === a.length - 1;
    if (g && b.parent) {
      break;
    }
    c = yb(c, a[e]);
    d = eb(d + "/" + a[e]);
    c.U && (!g || g && b.fa) && (c = c.U.root);
    if (!g || b.P) {
      for (g = 0; 40960 === (c.mode & 61440);) {
        if (c = Ib(d), d = jb(fb(d), c), c = T(d, {$:b.$ + 1}).node, 40 < g++) {
          throw new R(32);
        }
      }
    }
  }
  return {path:d, node:c};
}, Jb = a => {
  for (var b;;) {
    if (a === a.parent) {
      return a = a.B.ja, b ? "/" !== a[a.length - 1] ? `${a}/${b}` : a + b : a;
    }
    b = b ? `${a.name}/${b}` : a.name;
    a = a.parent;
  }
}, Kb = (a, b) => {
  for (var c = 0, d = 0; d < b.length; d++) {
    c = (c << 5) - c + b.charCodeAt(d) | 0;
  }
  return (a + c >>> 0) % Hb.length;
}, yb = (a, b) => {
  var c;
  if (c = (c = Lb(a, "x")) ? c : a.h.M ? 0 : 2) {
    throw new R(c, a);
  }
  for (c = Hb[Kb(a.id, b)]; c; c = c.Ia) {
    var d = c.name;
    if (c.parent.id === a.id && d === b) {
      return c;
    }
  }
  return a.h.M(a, b);
}, wb = (a, b, c, d) => {
  r("object" == typeof a);
  a = new Mb(a, b, c, d);
  b = Kb(a.parent.id, a.name);
  a.Ia = Hb[b];
  return Hb[b] = a;
}, Nb = a => {
  var b = ["r", "w", "rw"][a & 3];
  a & 512 && (b += "w");
  return b;
}, Lb = (a, b) => {
  if (Ma) {
    return 0;
  }
  if (!b.includes("r") || a.mode & 292) {
    if (b.includes("w") && !(a.mode & 146) || b.includes("x") && !(a.mode & 73)) {
      return 2;
    }
  } else {
    return 2;
  }
  return 0;
}, Ob = (a, b) => {
  try {
    return yb(a, b), 20;
  } catch (c) {
  }
  return Lb(a, "wx");
}, Pb = () => {
  for (var a = 0; 4096 >= a; a++) {
    if (!Fb[a]) {
      return a;
    }
  }
  throw new R(33);
}, U = a => {
  a = Fb[a];
  if (!a) {
    throw new R(8);
  }
  return a;
}, Rb = (a, b = -1) => {
  Qb || (Qb = function() {
    this.S = {};
  }, Qb.prototype = {}, Object.defineProperties(Qb.prototype, {object:{get:function() {
    return this.node;
  }, set:function(c) {
    this.node = c;
  }}, flags:{get:function() {
    return this.S.flags;
  }, set:function(c) {
    this.S.flags = c;
  },}, position:{get:function() {
    return this.S.position;
  }, set:function(c) {
    this.S.position = c;
  },},}));
  a = Object.assign(new Qb(), a);
  -1 == b && (b = Pb());
  a.o = b;
  return Fb[b] = a;
}, vb = {open:a => {
  a.i = Eb[a.node.N].i;
  a.i.open && a.i.open(a);
}, I:() => {
  throw new R(70);
}}, rb = (a, b) => {
  Eb[a] = {i:b};
}, Sb = (a, b) => {
  if ("string" == typeof a) {
    throw a;
  }
  var c = "/" === b, d = !b;
  if (c && Db) {
    throw new R(10);
  }
  if (!c && !d) {
    var e = T(b, {fa:!1});
    b = e.path;
    e = e.node;
    if (e.U) {
      throw new R(10);
    }
    if (16384 !== (e.mode & 61440)) {
      throw new R(54);
    }
  }
  b = {type:a, sb:{}, ja:b, Ha:[]};
  a = a.B(b);
  a.B = b;
  b.root = a;
  c ? Db = a : e && (e.U = b, e.B && e.B.Ha.push(b));
}, V = (a, b, c) => {
  var d = T(a, {parent:!0}).node;
  a = gb(a);
  if (!a || "." === a || ".." === a) {
    throw new R(28);
  }
  var e = Ob(d, a);
  if (e) {
    throw new R(e);
  }
  if (!d.h.T) {
    throw new R(63);
  }
  return d.h.T(d, a, b, c);
}, Tb = (a, b, c) => {
  "undefined" == typeof c && (c = b, b = 438);
  V(a, b | 8192, c);
}, Ka = (a, b) => {
  if (!jb(a)) {
    throw new R(44);
  }
  var c = T(b, {parent:!0}).node;
  if (!c) {
    throw new R(44);
  }
  b = gb(b);
  var d = Ob(c, b);
  if (d) {
    throw new R(d);
  }
  if (!c.h.V) {
    throw new R(63);
  }
  c.h.V(c, b, a);
}, Ib = a => {
  a = T(a).node;
  if (!a) {
    throw new R(44);
  }
  if (!a.h.O) {
    throw new R(28);
  }
  return jb(Jb(a.parent), a.h.O(a));
}, La = (a, b, c) => {
  if ("" === a) {
    throw new R(44);
  }
  if ("string" == typeof b) {
    var d = {r:0, "r+":2, w:577, "w+":578, a:1089, "a+":1090,}[b];
    if ("undefined" == typeof d) {
      throw Error(`Unknown file open mode: ${b}`);
    }
    b = d;
  }
  c = b & 64 ? ("undefined" == typeof c ? 438 : c) & 4095 | 32768 : 0;
  if ("object" == typeof a) {
    var e = a;
  } else {
    a = eb(a);
    try {
      e = T(a, {P:!(b & 131072)}).node;
    } catch (g) {
    }
  }
  d = !1;
  if (b & 64) {
    if (e) {
      if (b & 128) {
        throw new R(20);
      }
    } else {
      e = V(a, c, 0), d = !0;
    }
  }
  if (!e) {
    throw new R(44);
  }
  8192 === (e.mode & 61440) && (b &= -513);
  if (b & 65536 && 16384 !== (e.mode & 61440)) {
    throw new R(54);
  }
  if (!d && (c = e ? 40960 === (e.mode & 61440) ? 32 : 16384 === (e.mode & 61440) && ("r" !== Nb(b) || b & 512) ? 31 : Lb(e, Nb(b)) : 44)) {
    throw new R(c);
  }
  if (b & 512 && !d) {
    c = e;
    c = "string" == typeof c ? T(c, {P:!0}).node : c;
    if (!c.h.v) {
      throw new R(63);
    }
    if (16384 === (c.mode & 61440)) {
      throw new R(31);
    }
    if (32768 !== (c.mode & 61440)) {
      throw new R(28);
    }
    if (d = Lb(c, "w")) {
      throw new R(d);
    }
    c.h.v(c, {size:0, timestamp:Date.now()});
  }
  b &= -131713;
  e = Rb({node:e, path:Jb(e), flags:b, seekable:!0, position:0, i:e.i, Ya:[], error:!1});
  e.i.open && e.i.open(e);
  !f.logReadFiles || b & 1 || (Ub || (Ub = {}), a in Ub || (Ub[a] = 1));
  return e;
}, Vb = (a, b, c) => {
  if (null === a.o) {
    throw new R(8);
  }
  if (!a.seekable || !a.i.I) {
    throw new R(70);
  }
  if (0 != c && 1 != c && 2 != c) {
    throw new R(28);
  }
  a.position = a.i.I(a, b, c);
  a.Ya = [];
}, Ia = () => {
  R || (R = function(a, b) {
    this.name = "ErrnoError";
    this.node = b;
    this.Qa = function(c) {
      this.F = c;
      for (var d in Bb) {
        if (Bb[d] === c) {
          this.code = d;
          break;
        }
      }
    };
    this.Qa(a);
    this.message = Ab[a];
    this.stack && (Object.defineProperty(this, "stack", {value:Error().stack, writable:!0}), this.stack = Cb(this.stack));
  }, R.prototype = Error(), R.prototype.constructor = R, [44].forEach(a => {
    xb[a] = new R(a);
    xb[a].stack = "<generic error, no stack>";
  }));
}, Ha, Ja = (a, b, c) => {
  a = eb("/dev/" + a);
  var d = zb(!!b, !!c);
  Wb || (Wb = 64);
  var e = Wb++ << 8 | 0;
  rb(e, {open:g => {
    g.seekable = !1;
  }, close:() => {
    c && c.buffer && c.buffer.length && c(10);
  }, read:(g, k, t, B) => {
    for (var n = 0, y = 0; y < B; y++) {
      try {
        var D = b();
      } catch (E) {
        throw new R(29);
      }
      if (void 0 === D && 0 === n) {
        throw new R(6);
      }
      if (null === D || void 0 === D) {
        break;
      }
      n++;
      k[t + y] = D;
    }
    n && (g.node.timestamp = Date.now());
    return n;
  }, write:(g, k, t, B) => {
    for (var n = 0; n < B; n++) {
      try {
        c(k[t + n]);
      } catch (y) {
        throw new R(29);
      }
    }
    B && (g.node.timestamp = Date.now());
    return n;
  }});
  Tb(a, d, e);
}, Wb, W = {}, Qb, Ub, X = (a, b) => {
  r("number" == typeof a);
  return a ? ob(oa, a, b) : "";
}, Xb = void 0;
function Y() {
  r(void 0 != Xb);
  Xb += 4;
  return H[Xb - 4 >> 2];
}
function Yb(a) {
  if (p) {
    return Z(1, 1, a);
  }
  C = a;
  if (!Fa()) {
    Q.Sa();
    if (f.onExit) {
      f.onExit(a);
    }
    A = !0;
  }
  da(a, new ka(a));
}
var ac = (a, b) => {
  C = a;
  Zb();
  if (p) {
    throw r(!b), $b(a), "unwind";
  }
  Fa() && !b && (b = `program exited (with status: ${a}), but keepRuntimeAlive() is set (counter=${Ea}) due to an async operation, so halting execution but not exiting the runtime or preventing further async execution (you can use emscripten_force_exit, if you want to force a true shutdown)`, ba(b), v(b));
  Yb(a);
}, wa = a => {
  r("number" === typeof a);
  return "0x" + a.toString(16).padStart(8, "0");
}, cc = a => {
  a instanceof ka || "unwind" == a || (va(), a instanceof WebAssembly.RuntimeError && 0 >= bc() && v("Stack overflow detected.  You can try increasing -sSTACK_SIZE (currently set to 65536)"), da(1, a));
}, Q = {H:[], J:[], qa:[], D:{}, Ja:1, lb:function() {
}, ya:function() {
  p ? Q.Aa() : Q.za();
}, za:function() {
  for (var a = navigator.hardwareConcurrency + 3; a--;) {
    Q.ca();
  }
  za.unshift(() => {
    Qa("loading-workers");
    Q.Ga(() => Ra("loading-workers"));
  });
}, Aa:function() {
  Q.receiveObjectTransfer = Q.Na;
  Q.threadInitTLS = Q.pa;
  Q.setExitStatus = Q.oa;
  noExitRuntime = !1;
}, oa:function(a) {
  C = a;
}, vb:["$terminateWorker"], Sa:function() {
  r(!p, "Internal Error! terminateAllThreads() can only ever be called from main application thread!");
  for (var a of Q.J) {
    ab(a);
  }
  for (a of Q.H) {
    ab(a);
  }
  Q.H = [];
  Q.J = [];
  Q.D = [];
}, Pa:function(a) {
  var b = a.s;
  delete Q.D[b];
  Q.H.push(a);
  Q.J.splice(Q.J.indexOf(a), 1);
  a.s = 0;
  dc(b);
}, Na:function() {
}, pa:function() {
  Q.qa.forEach(a => a());
}, ha:a => new Promise(b => {
  a.onmessage = g => {
    g = g.data;
    var k = g.cmd;
    a.s && (Q.va = a.s);
    if (g.targetThread && g.targetThread != ec()) {
      var t = Q.D[g.ub];
      t ? t.postMessage(g, g.transferList) : v('Internal error! Worker sent a message "' + k + '" to target pthread ' + g.targetThread + ", but that thread no longer exists!");
    } else {
      if ("checkMailbox" === k) {
        fc();
      } else if ("spawnThread" === k) {
        cb(g);
      } else if ("cleanupThread" === k) {
        bb(g.thread);
      } else if ("killThread" === k) {
        g = g.thread, r(!p, "Internal Error! killThread() can only ever be called from main application thread!"), r(g, "Internal Error! Null pthread_ptr in killThread!"), k = Q.D[g], delete Q.D[g], ab(k), dc(g), Q.J.splice(Q.J.indexOf(k), 1), k.s = 0;
      } else if ("cancelThread" === k) {
        g = g.thread, r(!p, "Internal Error! cancelThread() can only ever be called from main application thread!"), r(g, "Internal Error! Null pthread_ptr in cancelThread!"), Q.D[g].postMessage({cmd:"cancel"});
      } else if ("loaded" === k) {
        a.loaded = !0, b(a);
      } else if ("alert" === k) {
        alert("Thread " + g.threadId + ": " + g.text);
      } else if ("setimmediate" === g.target) {
        a.postMessage(g);
      } else if ("callHandler" === k) {
        f[g.handler](...g.args);
      } else {
        k && v("worker sent an unknown command " + k);
      }
    }
    Q.va = void 0;
  };
  a.onerror = g => {
    var k = "worker sent an error!";
    a.s && (k = "Pthread " + wa(a.s) + " sent an error!");
    v(k + " " + g.filename + ":" + g.lineno + ": " + g.message);
    throw g;
  };
  r(z instanceof WebAssembly.Memory, "WebAssembly memory should have been loaded by now!");
  r(na instanceof WebAssembly.Module, "WebAssembly Module should have been loaded by now!");
  var c = [], d = ["onExit", "onAbort", "print", "printErr",], e;
  for (e of d) {
    f.hasOwnProperty(e) && c.push(e);
  }
  a.sa = Q.Ja++;
  a.postMessage({cmd:"load", handlers:c, urlOrBlob:f.mainScriptUrlOrBlob, wasmMemory:z, wasmModule:na, workerID:a.sa,});
}), Ga:function(a) {
  if (p) {
    return a();
  }
  Promise.all(Q.H.map(Q.ha)).then(a);
}, ca:function() {
  if (f.locateFile) {
    var a = ia("magpie_wasm.worker.js");
    a = new Worker(a);
  } else {
    a = new Worker(new URL("magpie_wasm.worker.js", import.meta.url));
  }
  Q.H.push(a);
}, xa:function() {
  0 == Q.H.length && (v("Tried to spawn a new thread, but the thread pool is exhausted.\nThis might result in a deadlock unless some threads eventually exit or the code explicitly breaks out to the event loop.\nIf you want to increase the pool size, use setting `-sPTHREAD_POOL_SIZE=...`.\nIf you want to throw an explicit error instead of the risk of deadlocking in those cases, use setting `-sPTHREAD_POOL_SIZE_STRICT=2`."), Q.ca(), Q.ha(Q.H[0]));
  return Q.H.pop();
}};
f.PThread = Q;
var Na = a => {
  for (; 0 < a.length;) {
    a.shift()(f);
  }
};
f.establishStackSpace = function() {
  var a = ec(), b = H[a + 52 >> 2];
  a = b - H[a + 56 >> 2];
  r(0 != b);
  r(0 != a);
  r(b > a, "stackHigh must be higher then stackLow");
  gc(b, a);
  hc(b);
  ta();
};
function $b(a) {
  if (p) {
    return Z(2, 0, a);
  }
  ac(a);
}
var ic = [];
f.invokeEntryPoint = function(a, b) {
  var c = ic[a];
  c || (a >= ic.length && (ic.length = a + 1), ic[a] = c = sa.get(a));
  r(sa.get(a) == c, "JavaScript-side Wasm function table mirror is out of date!");
  a = c(b);
  va();
  Fa() ? Q.oa(a) : jc(a);
};
var P = a => {
  kc || (kc = {});
  kc[a] || (kc[a] = 1, v(a));
}, kc;
function lc(a, b, c, d) {
  return p ? Z(3, 1, a, b, c, d) : mc(a, b, c, d);
}
function mc(a, b, c, d) {
  if ("undefined" == typeof SharedArrayBuffer) {
    return v("Current environment does not support SharedArrayBuffer, pthreads are not available!"), 6;
  }
  var e = [];
  if (p && 0 === e.length) {
    return lc(a, b, c, d);
  }
  a = {Ra:c, s:a, ta:d, Xa:e,};
  return p ? (a.jb = "spawnThread", postMessage(a, e), 0) : cb(a);
}
function nc(a, b, c) {
  if (p) {
    return Z(4, 1, a, b, c);
  }
  Xb = c;
  try {
    var d = U(a);
    switch(b) {
      case 0:
        var e = Y();
        return 0 > e ? -28 : Rb(d, e).o;
      case 1:
      case 2:
        return 0;
      case 3:
        return d.flags;
      case 4:
        return e = Y(), d.flags |= e, 0;
      case 5:
        return e = Y(), pa[e + 0 >> 1] = 2, 0;
      case 6:
      case 7:
        return 0;
      case 16:
      case 8:
        return -28;
      case 9:
        return H[oc() >> 2] = 28, -1;
      default:
        return -28;
    }
  } catch (g) {
    if ("undefined" == typeof W || "ErrnoError" !== g.name) {
      throw g;
    }
    return -g.F;
  }
}
function pc(a, b, c) {
  if (p) {
    return Z(5, 1, a, b, c);
  }
  Xb = c;
  try {
    var d = U(a);
    switch(b) {
      case 21509:
        return d.j ? 0 : -59;
      case 21505:
        if (!d.j) {
          return -59;
        }
        if (d.j.C.Ca) {
          b = [3, 28, 127, 21, 4, 0, 1, 0, 17, 19, 26, 0, 18, 15, 23, 22, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,];
          var e = Y();
          H[e >> 2] = 25856;
          H[e + 4 >> 2] = 5;
          H[e + 8 >> 2] = 191;
          H[e + 12 >> 2] = 35387;
          for (var g = 0; 32 > g; g++) {
            G[e + g + 17 >> 0] = b[g] || 0;
          }
        }
        return 0;
      case 21510:
      case 21511:
      case 21512:
        return d.j ? 0 : -59;
      case 21506:
      case 21507:
      case 21508:
        if (!d.j) {
          return -59;
        }
        if (d.j.C.Da) {
          for (e = Y(), b = [], g = 0; 32 > g; g++) {
            b.push(G[e + g + 17 >> 0]);
          }
        }
        return 0;
      case 21519:
        if (!d.j) {
          return -59;
        }
        e = Y();
        return H[e >> 2] = 0;
      case 21520:
        return d.j ? -28 : -59;
      case 21531:
        e = Y();
        if (!d.i.Ba) {
          throw new R(59);
        }
        return d.i.Ba(d, b, e);
      case 21523:
        if (!d.j) {
          return -59;
        }
        d.j.C.Ea && (g = [24, 80], e = Y(), pa[e >> 1] = g[0], pa[e + 2 >> 1] = g[1]);
        return 0;
      case 21524:
        return d.j ? 0 : -59;
      case 21515:
        return d.j ? 0 : -59;
      default:
        return -28;
    }
  } catch (k) {
    if ("undefined" == typeof W || "ErrnoError" !== k.name) {
      throw k;
    }
    return -k.F;
  }
}
function qc(a, b, c, d) {
  if (p) {
    return Z(6, 1, a, b, c, d);
  }
  Xb = d;
  try {
    b = X(b);
    var e = b;
    if ("/" === e.charAt(0)) {
      b = e;
    } else {
      var g = -100 === a ? "/" : U(a).path;
      if (0 == e.length) {
        throw new R(44);
      }
      b = eb(g + "/" + e);
    }
    var k = d ? Y() : 0;
    return La(b, c, k).o;
  } catch (t) {
    if ("undefined" == typeof W || "ErrnoError" !== t.name) {
      throw t;
    }
    return -t.F;
  }
}
var rc = a => {
  if (A) {
    v("user callback triggered after runtime exited or application aborted.  Ignoring.");
  } else {
    try {
      if (a(), !Fa()) {
        try {
          p ? jc(C) : ac(C);
        } catch (b) {
          cc(b);
        }
      }
    } catch (b) {
      cc(b);
    }
  }
};
function sc(a) {
  if ("function" === typeof Atomics.$a) {
    var b = Atomics.$a(H, a >> 2, a);
    r(b.async);
    b.value.then(fc);
    Atomics.store(H, a + 128 >> 2, 1);
  }
}
f.__emscripten_thread_mailbox_await = sc;
function fc() {
  var a = ec();
  a && (sc(a), rc(() => tc()));
}
f.checkMailbox = fc;
var uc = a => 0 === a % 4 && (0 !== a % 100 || 0 === a % 400), vc = [0, 31, 60, 91, 121, 152, 182, 213, 244, 274, 305, 335], wc = [0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334], xc = (a, b, c) => {
  r("number" == typeof c, "stringToUTF8(str, outPtr, maxBytesToWrite) is missing the third parameter that specifies the length of the output buffer!");
  lb(a, oa, b, c);
}, zc = a => {
  var b = kb(a) + 1, c = yc(b);
  c && xc(a, c, b);
  return c;
}, Bc = a => {
  var b = Ac();
  a = a();
  hc(b);
  return a;
};
function Z(a, b) {
  var c = arguments.length - 2, d = arguments;
  if (19 < c) {
    throw "proxyToMainThread: Too many arguments " + c + " to proxied function idx=" + a + ", maximum supported is 19";
  }
  return Bc(() => {
    for (var e = Cc(8 * c), g = e >> 3, k = 0; k < c; k++) {
      qa[g + k] = d[2 + k];
    }
    return Dc(a, c, e, b);
  });
}
var Ec = [];
function Fc(a) {
  if (p) {
    return Z(7, 1, a);
  }
  try {
    var b = U(a);
    if (null === b.o) {
      throw new R(8);
    }
    b.Y && (b.Y = null);
    try {
      b.i.close && b.i.close(b);
    } catch (c) {
      throw c;
    } finally {
      Fb[b.o] = null;
    }
    b.o = null;
    return 0;
  } catch (c) {
    if ("undefined" == typeof W || "ErrnoError" !== c.name) {
      throw c;
    }
    return c.F;
  }
}
function Gc(a, b, c, d) {
  if (p) {
    return Z(8, 1, a, b, c, d);
  }
  try {
    a: {
      var e = U(a);
      a = b;
      for (var g, k = b = 0; k < c; k++) {
        var t = I[a >> 2], B = I[a + 4 >> 2];
        a += 8;
        var n = e, y = t, D = B, E = g;
        if (0 > D || 0 > E) {
          throw new R(28);
        }
        if (null === n.o) {
          throw new R(8);
        }
        if (1 === (n.flags & 2097155)) {
          throw new R(8);
        }
        if (16384 === (n.node.mode & 61440)) {
          throw new R(31);
        }
        if (!n.i.read) {
          throw new R(28);
        }
        var h = "undefined" != typeof E;
        if (!h) {
          E = n.position;
        } else if (!n.seekable) {
          throw new R(70);
        }
        var u = n.i.read(n, G, y, D, E);
        h || (n.position += u);
        var x = u;
        if (0 > x) {
          var F = -1;
          break a;
        }
        b += x;
        if (x < B) {
          break;
        }
        "undefined" !== typeof g && (g += x);
      }
      F = b;
    }
    I[d >> 2] = F;
    return 0;
  } catch (M) {
    if ("undefined" == typeof W || "ErrnoError" !== M.name) {
      throw M;
    }
    return M.F;
  }
}
function Hc(a, b, c, d, e) {
  if (p) {
    return Z(9, 1, a, b, c, d, e);
  }
  try {
    r(b == b >>> 0 || b == (b | 0));
    r(c === (c | 0));
    var g = c + 2097152 >>> 0 < 4194305 - !!b ? (b >>> 0) + 4294967296 * c : NaN;
    if (isNaN(g)) {
      return 61;
    }
    var k = U(a);
    Vb(k, g, d);
    Ya = [k.position >>> 0, (Xa = k.position, 1.0 <= +Math.abs(Xa) ? 0.0 < Xa ? +Math.floor(Xa / 4294967296.0) >>> 0 : ~~+Math.ceil((Xa - +(~~Xa >>> 0)) / 4294967296.0) >>> 0 : 0)];
    H[e >> 2] = Ya[0];
    H[e + 4 >> 2] = Ya[1];
    k.Y && 0 === g && 0 === d && (k.Y = null);
    return 0;
  } catch (t) {
    if ("undefined" == typeof W || "ErrnoError" !== t.name) {
      throw t;
    }
    return t.F;
  }
}
function Ic(a, b, c, d) {
  if (p) {
    return Z(10, 1, a, b, c, d);
  }
  try {
    a: {
      var e = U(a);
      a = b;
      for (var g, k = b = 0; k < c; k++) {
        var t = I[a >> 2], B = I[a + 4 >> 2];
        a += 8;
        var n = e, y = t, D = B, E = g;
        if (0 > D || 0 > E) {
          throw new R(28);
        }
        if (null === n.o) {
          throw new R(8);
        }
        if (0 === (n.flags & 2097155)) {
          throw new R(8);
        }
        if (16384 === (n.node.mode & 61440)) {
          throw new R(31);
        }
        if (!n.i.write) {
          throw new R(28);
        }
        n.seekable && n.flags & 1024 && Vb(n, 0, 2);
        var h = "undefined" != typeof E;
        if (!h) {
          E = n.position;
        } else if (!n.seekable) {
          throw new R(70);
        }
        var u = n.i.write(n, G, y, D, E, void 0);
        h || (n.position += u);
        var x = u;
        if (0 > x) {
          var F = -1;
          break a;
        }
        b += x;
        "undefined" !== typeof g && (g += x);
      }
      F = b;
    }
    I[d >> 2] = F;
    return 0;
  } catch (M) {
    if ("undefined" == typeof W || "ErrnoError" !== M.name) {
      throw M;
    }
    return M.F;
  }
}
var Jc = [31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31], Kc = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31], Lc = (a, b) => {
  r(0 <= a.length, "writeArrayToMemory array must have a length (should be an array or typed array)");
  G.set(a, b);
};
function Mc(a) {
  var b = f["_" + a];
  r(b, "Cannot call unknown function " + a + ", make sure it is exported");
  return b;
}
function Nc(a, b, c, d) {
  var e = {string:n => {
    var y = 0;
    if (null !== n && void 0 !== n && 0 !== n) {
      y = kb(n) + 1;
      var D = Cc(y);
      xc(n, D, y);
      y = D;
    }
    return y;
  }, array:n => {
    var y = Cc(n.length);
    Lc(n, y);
    return y;
  }};
  a = Mc(a);
  var g = [], k = 0;
  r("array" !== b, 'Return type should not be "array".');
  if (d) {
    for (var t = 0; t < d.length; t++) {
      var B = e[c[t]];
      B ? (0 === k && (k = Ac()), g[t] = B(d[t])) : g[t] = d[t];
    }
  }
  c = a.apply(null, g);
  return c = function(n) {
    0 !== k && hc(k);
    return "string" === b ? X(n) : "boolean" === b ? !!n : n;
  }(c);
}
Q.ya();
function Mb(a, b, c, d) {
  a || (a = this);
  this.parent = a;
  this.B = a.B;
  this.U = null;
  this.id = Gb++;
  this.name = b;
  this.mode = c;
  this.h = {};
  this.i = {};
  this.N = d;
}
Object.defineProperties(Mb.prototype, {read:{get:function() {
  return 365 === (this.mode & 365);
}, set:function(a) {
  a ? this.mode |= 365 : this.mode &= -366;
}}, write:{get:function() {
  return 146 === (this.mode & 146);
}, set:function(a) {
  a ? this.mode |= 146 : this.mode &= -147;
}}});
Ia();
Hb = Array(4096);
Sb(S, "/");
V("/tmp", 16895, 0);
V("/home", 16895, 0);
V("/home/web_user", 16895, 0);
(() => {
  V("/dev", 16895, 0);
  rb(259, {read:() => 0, write:(d, e, g, k) => k,});
  Tb("/dev/null", 259);
  qb(1280, tb);
  qb(1536, ub);
  Tb("/dev/tty", 1280);
  Tb("/dev/tty1", 1536);
  var a = new Uint8Array(1024), b = 0, c = () => {
    0 === b && (b = ib(a).byteLength);
    return a[--b];
  };
  Ja("random", c);
  Ja("urandom", c);
  V("/dev/shm", 16895, 0);
  V("/dev/shm/tmp", 16895, 0);
})();
(() => {
  V("/proc", 16895, 0);
  var a = V("/proc/self", 16895, 0);
  V("/proc/self/fd", 16895, 0);
  Sb({B:() => {
    var b = wb(a, "fd", 16895, 73);
    b.h = {M:(c, d) => {
      var e = U(+d);
      c = {parent:null, B:{ja:"fake"}, h:{O:() => e.path},};
      return c.parent = c;
    }};
    return b;
  }}, "/proc/self/fd");
})();
Bb = {EPERM:63, ENOENT:44, ESRCH:71, EINTR:27, EIO:29, ENXIO:60, E2BIG:1, ENOEXEC:45, EBADF:8, ECHILD:12, EAGAIN:6, EWOULDBLOCK:6, ENOMEM:48, EACCES:2, EFAULT:21, ENOTBLK:105, EBUSY:10, EEXIST:20, EXDEV:75, ENODEV:43, ENOTDIR:54, EISDIR:31, EINVAL:28, ENFILE:41, EMFILE:33, ENOTTY:59, ETXTBSY:74, EFBIG:22, ENOSPC:51, ESPIPE:70, EROFS:69, EMLINK:34, EPIPE:64, EDOM:18, ERANGE:68, ENOMSG:49, EIDRM:24, ECHRNG:106, EL2NSYNC:156, EL3HLT:107, EL3RST:108, ELNRNG:109, EUNATCH:110, ENOCSI:111, EL2HLT:112, EDEADLK:16, 
ENOLCK:46, EBADE:113, EBADR:114, EXFULL:115, ENOANO:104, EBADRQC:103, EBADSLT:102, EDEADLOCK:16, EBFONT:101, ENOSTR:100, ENODATA:116, ETIME:117, ENOSR:118, ENONET:119, ENOPKG:120, EREMOTE:121, ENOLINK:47, EADV:122, ESRMNT:123, ECOMM:124, EPROTO:65, EMULTIHOP:36, EDOTDOT:125, EBADMSG:9, ENOTUNIQ:126, EBADFD:127, EREMCHG:128, ELIBACC:129, ELIBBAD:130, ELIBSCN:131, ELIBMAX:132, ELIBEXEC:133, ENOSYS:52, ENOTEMPTY:55, ENAMETOOLONG:37, ELOOP:32, EOPNOTSUPP:138, EPFNOSUPPORT:139, ECONNRESET:15, ENOBUFS:42, 
EAFNOSUPPORT:5, EPROTOTYPE:67, ENOTSOCK:57, ENOPROTOOPT:50, ESHUTDOWN:140, ECONNREFUSED:14, EADDRINUSE:3, ECONNABORTED:13, ENETUNREACH:40, ENETDOWN:38, ETIMEDOUT:73, EHOSTDOWN:142, EHOSTUNREACH:23, EINPROGRESS:26, EALREADY:7, EDESTADDRREQ:17, EMSGSIZE:35, EPROTONOSUPPORT:66, ESOCKTNOSUPPORT:137, EADDRNOTAVAIL:4, ENETRESET:39, EISCONN:30, ENOTCONN:53, ETOOMANYREFS:141, EUSERS:136, EDQUOT:19, ESTALE:72, ENOTSUP:138, ENOMEDIUM:148, EILSEQ:25, EOVERFLOW:61, ECANCELED:11, ENOTRECOVERABLE:56, EOWNERDEAD:62, 
ESTRPIPE:135,};
var Oc = [null, Yb, $b, lc, nc, pc, qc, Fc, Gc, Hc, Ic], Qc = {__assert_fail:(a, b, c, d) => {
  l(`Assertion failed: ${X(a)}, at: ` + [b ? X(b) : "unknown filename", c, d ? X(d) : "unknown function"]);
}, __emscripten_init_main_thread_js:function(a) {
  Pc(a, !m, 1, !ea, 65536,);
  Q.pa();
}, __emscripten_thread_cleanup:function(a) {
  p ? postMessage({cmd:"cleanupThread", thread:a}) : bb(a);
}, __pthread_create_js:mc, __syscall_fcntl64:nc, __syscall_ioctl:pc, __syscall_openat:qc, _emscripten_get_now_is_monotonic:() => !0, _emscripten_notify_mailbox_postmessage:function(a, b) {
  a == b ? setTimeout(() => fc()) : p ? postMessage({targetThread:a, cmd:"checkMailbox"}) : (b = Q.D[a]) ? b.postMessage({cmd:"checkMailbox"}) : v("Cannot send message to thread with ID " + a + ", unknown thread ID!");
}, _emscripten_set_offscreencanvas_size:function() {
  v("emscripten_set_offscreencanvas_size: Build with -sOFFSCREENCANVAS_SUPPORT=1 to enable transferring canvases to pthreads.");
  return -1;
}, _emscripten_thread_mailbox_await:sc, _emscripten_thread_set_strongref:function() {
}, _localtime_js:(a, b) => {
  a = new Date(1000 * (I[a >> 2] + 4294967296 * H[a + 4 >> 2]));
  H[b >> 2] = a.getSeconds();
  H[b + 4 >> 2] = a.getMinutes();
  H[b + 8 >> 2] = a.getHours();
  H[b + 12 >> 2] = a.getDate();
  H[b + 16 >> 2] = a.getMonth();
  H[b + 20 >> 2] = a.getFullYear() - 1900;
  H[b + 24 >> 2] = a.getDay();
  H[b + 28 >> 2] = (uc(a.getFullYear()) ? vc : wc)[a.getMonth()] + a.getDate() - 1 | 0;
  H[b + 36 >> 2] = -(60 * a.getTimezoneOffset());
  var c = (new Date(a.getFullYear(), 6, 1)).getTimezoneOffset(), d = (new Date(a.getFullYear(), 0, 1)).getTimezoneOffset();
  H[b + 32 >> 2] = (c != d && a.getTimezoneOffset() == Math.min(d, c)) | 0;
}, _tzset_js:(a, b, c) => {
  function d(B) {
    return (B = B.toTimeString().match(/\(([A-Za-z ]+)\)$/)) ? B[1] : "GMT";
  }
  var e = (new Date()).getFullYear(), g = new Date(e, 0, 1), k = new Date(e, 6, 1);
  e = g.getTimezoneOffset();
  var t = k.getTimezoneOffset();
  I[a >> 2] = 60 * Math.max(e, t);
  H[b >> 2] = Number(e != t);
  a = d(g);
  b = d(k);
  a = zc(a);
  b = zc(b);
  t < e ? (I[c >> 2] = a, I[c + 4 >> 2] = b) : (I[c >> 2] = b, I[c + 4 >> 2] = a);
}, abort:() => {
  l("native code called abort()");
}, emscripten_check_blocking_allowed:function() {
  m || (P("Blocking on the main thread is very dangerous, see https://emscripten.org/docs/porting/pthreads.html#blocking-on-the-main-browser-thread"), l("Blocking on the main thread is not allowed by default. See https://emscripten.org/docs/porting/pthreads.html#blocking-on-the-main-browser-thread"));
}, emscripten_date_now:function() {
  return Date.now();
}, emscripten_exit_with_live_runtime:() => {
  Ea += 1;
  throw "unwind";
}, emscripten_get_now:() => performance.timeOrigin + performance.now(), emscripten_receive_on_main_thread_js:function(a, b, c) {
  Ec.length = b;
  c >>= 3;
  for (var d = 0; d < b; d++) {
    Ec[d] = qa[c + d];
  }
  a = Oc[a];
  r(a.length == b, "Call args mismatch in emscripten_receive_on_main_thread_js");
  return a.apply(null, Ec);
}, emscripten_resize_heap:a => {
  l(`Cannot enlarge memory arrays to size ${a >>> 0} bytes (OOM). Either (1) compile with -sINITIAL_MEMORY=X with X higher than the current value ${G.length}, (2) compile with -sALLOW_MEMORY_GROWTH which allows increasing the size at runtime, or (3) if you want malloc to return NULL (0) instead of this abort, compile with -sABORTING_MALLOC=0`);
}, exit:ac, fd_close:Fc, fd_read:Gc, fd_seek:Hc, fd_write:Ic, memory:z || f.wasmMemory, strftime:(a, b, c, d) => {
  function e(h, u, x) {
    for (h = "number" == typeof h ? h.toString() : h || ""; h.length < u;) {
      h = x[0] + h;
    }
    return h;
  }
  function g(h, u) {
    return e(h, u, "0");
  }
  function k(h, u) {
    function x(M) {
      return 0 > M ? -1 : 0 < M ? 1 : 0;
    }
    var F;
    0 === (F = x(h.getFullYear() - u.getFullYear())) && 0 === (F = x(h.getMonth() - u.getMonth())) && (F = x(h.getDate() - u.getDate()));
    return F;
  }
  function t(h) {
    switch(h.getDay()) {
      case 0:
        return new Date(h.getFullYear() - 1, 11, 29);
      case 1:
        return h;
      case 2:
        return new Date(h.getFullYear(), 0, 3);
      case 3:
        return new Date(h.getFullYear(), 0, 2);
      case 4:
        return new Date(h.getFullYear(), 0, 1);
      case 5:
        return new Date(h.getFullYear() - 1, 11, 31);
      case 6:
        return new Date(h.getFullYear() - 1, 11, 30);
    }
  }
  function B(h) {
    var u = h.K;
    for (h = new Date((new Date(h.L + 1900, 0, 1)).getTime()); 0 < u;) {
      var x = h.getMonth(), F = (uc(h.getFullYear()) ? Jc : Kc)[x];
      if (u > F - h.getDate()) {
        u -= F - h.getDate() + 1, h.setDate(1), 11 > x ? h.setMonth(x + 1) : (h.setMonth(0), h.setFullYear(h.getFullYear() + 1));
      } else {
        h.setDate(h.getDate() + u);
        break;
      }
    }
    x = new Date(h.getFullYear() + 1, 0, 4);
    u = t(new Date(h.getFullYear(), 0, 4));
    x = t(x);
    return 0 >= k(u, h) ? 0 >= k(x, h) ? h.getFullYear() + 1 : h.getFullYear() : h.getFullYear() - 1;
  }
  var n = H[d + 40 >> 2];
  d = {Va:H[d >> 2], Ua:H[d + 4 >> 2], W:H[d + 8 >> 2], aa:H[d + 12 >> 2], X:H[d + 16 >> 2], L:H[d + 20 >> 2], A:H[d + 24 >> 2], K:H[d + 28 >> 2], wb:H[d + 32 >> 2], Ta:H[d + 36 >> 2], Wa:n ? X(n) : ""};
  c = X(c);
  n = {"%c":"%a %b %d %H:%M:%S %Y", "%D":"%m/%d/%y", "%F":"%Y-%m-%d", "%h":"%b", "%r":"%I:%M:%S %p", "%R":"%H:%M", "%T":"%H:%M:%S", "%x":"%m/%d/%y", "%X":"%H:%M:%S", "%Ec":"%c", "%EC":"%C", "%Ex":"%m/%d/%y", "%EX":"%H:%M:%S", "%Ey":"%y", "%EY":"%Y", "%Od":"%d", "%Oe":"%e", "%OH":"%H", "%OI":"%I", "%Om":"%m", "%OM":"%M", "%OS":"%S", "%Ou":"%u", "%OU":"%U", "%OV":"%V", "%Ow":"%w", "%OW":"%W", "%Oy":"%y",};
  for (var y in n) {
    c = c.replace(new RegExp(y, "g"), n[y]);
  }
  var D = "Sunday Monday Tuesday Wednesday Thursday Friday Saturday".split(" "), E = "January February March April May June July August September October November December".split(" ");
  n = {"%a":h => D[h.A].substring(0, 3), "%A":h => D[h.A], "%b":h => E[h.X].substring(0, 3), "%B":h => E[h.X], "%C":h => g((h.L + 1900) / 100 | 0, 2), "%d":h => g(h.aa, 2), "%e":h => e(h.aa, 2, " "), "%g":h => B(h).toString().substring(2), "%G":h => B(h), "%H":h => g(h.W, 2), "%I":h => {
    h = h.W;
    0 == h ? h = 12 : 12 < h && (h -= 12);
    return g(h, 2);
  }, "%j":h => {
    for (var u = 0, x = 0; x <= h.X - 1; u += (uc(h.L + 1900) ? Jc : Kc)[x++]) {
    }
    return g(h.aa + u, 3);
  }, "%m":h => g(h.X + 1, 2), "%M":h => g(h.Ua, 2), "%n":() => "\n", "%p":h => 0 <= h.W && 12 > h.W ? "AM" : "PM", "%S":h => g(h.Va, 2), "%t":() => "\t", "%u":h => h.A || 7, "%U":h => g(Math.floor((h.K + 7 - h.A) / 7), 2), "%V":h => {
    var u = Math.floor((h.K + 7 - (h.A + 6) % 7) / 7);
    2 >= (h.A + 371 - h.K - 2) % 7 && u++;
    if (u) {
      53 == u && (x = (h.A + 371 - h.K) % 7, 4 == x || 3 == x && uc(h.L) || (u = 1));
    } else {
      u = 52;
      var x = (h.A + 7 - h.K - 1) % 7;
      (4 == x || 5 == x && uc(h.L % 400 - 1)) && u++;
    }
    return g(u, 2);
  }, "%w":h => h.A, "%W":h => g(Math.floor((h.K + 7 - (h.A + 6) % 7) / 7), 2), "%y":h => (h.L + 1900).toString().substring(2), "%Y":h => h.L + 1900, "%z":h => {
    h = h.Ta;
    var u = 0 <= h;
    h = Math.abs(h) / 60;
    return (u ? "+" : "-") + String("0000" + (h / 60 * 100 + h % 60)).slice(-4);
  }, "%Z":h => h.Wa, "%%":() => "%"};
  c = c.replace(/%%/g, "\x00\x00");
  for (y in n) {
    c.includes(y) && (c = c.replace(new RegExp(y, "g"), n[y](d)));
  }
  c = c.replace(/\0\0/g, "%");
  y = mb(c, !1);
  if (y.length > b) {
    return 0;
  }
  Lc(y, a);
  return y.length - 1;
}};
(function() {
  function a(d, e) {
    d = d.exports;
    f.asm = d;
    Q.qa.push(f.asm._emscripten_tls_init);
    sa = f.asm.__indirect_function_table;
    r(sa, "table not found in wasm exports");
    Aa.unshift(f.asm.__wasm_call_ctors);
    na = e;
    Ra("wasm-instantiate");
    return d;
  }
  var b = {env:Qc, wasi_snapshot_preview1:Qc,};
  Qa("wasm-instantiate");
  var c = f;
  if (f.instantiateWasm) {
    try {
      return f.instantiateWasm(b, a);
    } catch (d) {
      v("Module.instantiateWasm callback failed with error: " + d), ba(d);
    }
  }
  Wa(b, function(d) {
    r(f === c, "the Module object should not be replaced during async compilation - perhaps the order of HTML elements is wrong?");
    c = null;
    a(d.instance, d.module);
  }).catch(ba);
  return {};
})();
var yc = f._malloc = N("malloc");
f._free = N("free");
f._precache_file_data = N("precache_file_data");
f._destroy_cache = N("destroy_cache");
var Rc = f._fflush = N("fflush");
f._score_play = N("score_play");
f._static_evaluation = N("static_evaluation");
f._process_ucgi_command_wasm = N("process_ucgi_command_wasm");
f._ucgi_search_status_wasm = N("ucgi_search_status_wasm");
f._ucgi_stop_search_wasm = N("ucgi_stop_search_wasm");
var Sc = f._main = N("main");
f.__emscripten_tls_init = N("_emscripten_tls_init");
var ec = f._pthread_self = function() {
  return (ec = f._pthread_self = f.asm.pthread_self).apply(null, arguments);
}, oc = N("__errno_location"), Pc = f.__emscripten_thread_init = N("_emscripten_thread_init");
f.__emscripten_thread_crashed = N("_emscripten_thread_crashed");
var Dc = N("_emscripten_run_in_main_runtime_thread_js");
function ua() {
  return (ua = f.asm.emscripten_stack_get_end).apply(null, arguments);
}
var dc = N("_emscripten_thread_free_data"), jc = f.__emscripten_thread_exit = N("_emscripten_thread_exit"), tc = f.__emscripten_check_mailbox = N("_emscripten_check_mailbox");
function Tc() {
  return (Tc = f.asm.emscripten_stack_init).apply(null, arguments);
}
function gc() {
  return (gc = f.asm.emscripten_stack_set_limits).apply(null, arguments);
}
var Ac = N("stackSave"), hc = N("stackRestore"), Cc = N("stackAlloc");
function bc() {
  return (bc = f.asm.emscripten_stack_get_current).apply(null, arguments);
}
f.dynCall_jiji = N("dynCall_jiji");
f.keepRuntimeAlive = Fa;
f.wasmMemory = z;
f.cwrap = function(a, b, c) {
  return function() {
    return Nc(a, b, c, arguments);
  };
};
f.UTF8ToString = X;
f.stringToNewUTF8 = zc;
f.ExitStatus = ka;
f.PThread = Q;
"growMemory inetPton4 inetNtop4 inetPton6 inetNtop6 readSockaddr writeSockaddr getHostByName traverseStack getCallstack emscriptenLog convertPCtoSourceLocation readEmAsmArgs jstoi_q jstoi_s getExecutableName listenOnce autoResumeAudioContext dynCallLegacy getDynCaller dynCall runtimeKeepalivePop safeSetTimeout asmjsMangle HandleAllocator getNativeTypeSize STACK_SIZE STACK_ALIGN POINTER_SIZE ASSERTIONS writeI53ToI64 writeI53ToI64Clamped writeI53ToI64Signaling writeI53ToU64Clamped writeI53ToU64Signaling readI53FromU64 convertI32PairToI53 convertU32PairToI53 uleb128Encode sigToWasmTypes generateFuncType convertJsFunctionToWasm getEmptyTableSlot updateTableMap getFunctionAddress addFunction removeFunction reallyNegative unSign strLen reSign formatString intArrayToString AsciiToString stringToAscii UTF16ToString stringToUTF16 lengthBytesUTF16 UTF32ToString stringToUTF32 lengthBytesUTF32 registerKeyEventCallback maybeCStringToJsString findEventTarget findCanvasEventTarget getBoundingClientRect fillMouseEventData registerMouseEventCallback registerWheelEventCallback registerUiEventCallback registerFocusEventCallback fillDeviceOrientationEventData registerDeviceOrientationEventCallback fillDeviceMotionEventData registerDeviceMotionEventCallback screenOrientation fillOrientationChangeEventData registerOrientationChangeEventCallback fillFullscreenChangeEventData registerFullscreenChangeEventCallback JSEvents_requestFullscreen JSEvents_resizeCanvasForFullscreen registerRestoreOldStyle hideEverythingExceptGivenElement restoreHiddenElements setLetterbox softFullscreenResizeWebGLRenderTarget doRequestFullscreen fillPointerlockChangeEventData registerPointerlockChangeEventCallback registerPointerlockErrorEventCallback requestPointerLock fillVisibilityChangeEventData registerVisibilityChangeEventCallback registerTouchEventCallback fillGamepadEventData registerGamepadEventCallback registerBeforeUnloadEventCallback fillBatteryEventData battery registerBatteryEventCallback setCanvasElementSizeCallingThread setCanvasElementSizeMainThread setCanvasElementSize getCanvasSizeCallingThread getCanvasSizeMainThread getCanvasElementSize jsStackTrace stackTrace getEnvStrings checkWasiClock wasiRightsToMuslOFlags wasiOFlagsToMuslOFlags createDyncallWrapper setImmediateWrapped clearImmediateWrapped polyfillSetImmediate getPromise makePromise idsToPromises makePromiseCallback ExceptionInfo setMainLoop getSocketFromFD getSocketAddress _setNetworkCallback heapObjectForWebGLType heapAccessShiftForWebGLHeap webgl_enable_ANGLE_instanced_arrays webgl_enable_OES_vertex_array_object webgl_enable_WEBGL_draw_buffers webgl_enable_WEBGL_multi_draw emscriptenWebGLGet computeUnpackAlignedImageSize colorChannelsInGlTextureFormat emscriptenWebGLGetTexPixelData __glGenObject emscriptenWebGLGetUniform webglGetUniformLocation webglPrepareUniformLocationsBeforeFirstUse webglGetLeftBracePos emscriptenWebGLGetVertexAttrib __glGetActiveAttribOrUniform writeGLArray emscripten_webgl_destroy_context_before_on_calling_thread registerWebGlEventCallback runAndAbortIfError SDL_unicode SDL_ttfContext SDL_audio GLFW_Window ALLOC_NORMAL ALLOC_STACK allocate writeStringToMemory writeAsciiToMemory".split(" ").forEach(function(a) {
  "undefined" === typeof globalThis || Object.getOwnPropertyDescriptor(globalThis, a) || Object.defineProperty(globalThis, a, {configurable:!0, get:function() {
    var b = "`" + a + "` is a library symbol and not included by default; add it to your library.js __deps or to DEFAULT_LIBRARY_FUNCS_TO_INCLUDE on the command line", c = a;
    c.startsWith("_") || (c = "$" + a);
    b += " (e.g. -sDEFAULT_LIBRARY_FUNCS_TO_INCLUDE='" + c + "')";
    Za(a) && (b += ". Alternatively, forcing filesystem support (-sFORCE_FILESYSTEM) can export this for you");
    P(b);
  }});
  $a(a);
});
"run addOnPreRun addOnInit addOnPreMain addOnExit addOnPostRun addRunDependency removeRunDependency FS_createFolder FS_createPath FS_createDataFile FS_createLazyFile FS_createLink FS_createDevice FS_unlink out err callMain abort stackAlloc stackSave stackRestore getTempRet0 setTempRet0 writeStackCookie checkStackCookie ptrToString zeroMemory exitJS getHeapMax abortOnCannotGrowMemory ENV MONTH_DAYS_REGULAR MONTH_DAYS_LEAP MONTH_DAYS_REGULAR_CUMULATIVE MONTH_DAYS_LEAP_CUMULATIVE isLeapYear ydayFromDate arraySum addDays ERRNO_CODES ERRNO_MESSAGES setErrNo DNS Protocols Sockets initRandomFill randomFill timers warnOnce UNWIND_CACHE readEmAsmArgsArray handleException runtimeKeepalivePush callUserCallback maybeExit asyncLoad alignMemory mmapAlloc readI53FromI64 convertI32PairToI53Checked getCFunc ccall freeTableIndexes functionsInTableMap setValue getValue PATH PATH_FS UTF8Decoder UTF8ArrayToString stringToUTF8Array stringToUTF8 lengthBytesUTF8 intArrayFromString UTF16Decoder stringToUTF8OnStack writeArrayToMemory JSEvents specialHTMLTargets currentFullscreenStrategy restoreOldWindowedStyle demangle demangleAll doReadv doWritev promiseMap uncaughtExceptionCount exceptionLast exceptionCaught Browser wget SYSCALLS preloadPlugins FS_createPreloadedFile FS_modeStringToFlags FS_getMode FS MEMFS TTY PIPEFS SOCKFS tempFixedLengthArray miniTempWebGLFloatBuffers miniTempWebGLIntBuffers GL emscripten_webgl_power_preferences AL GLUT EGL GLEW IDBStore SDL SDL_gfx GLFW allocateUTF8 allocateUTF8OnStack terminateWorker killThread cleanupThread registerTLSInit cancelThread spawnThread exitOnMainThread proxyToMainThread emscripten_receive_on_main_thread_js_callArgs invokeEntryPoint checkMailbox".split(" ").forEach($a);
var Uc;
Oa = function Vc() {
  Uc || Wc();
  Uc || (Oa = Vc);
};
function Wc() {
  function a() {
    if (!Uc && (Uc = !0, f.calledRun = !0, !A)) {
      Ga();
      va();
      p || Na(Ba);
      aa(f);
      if (f.onRuntimeInitialized) {
        f.onRuntimeInitialized();
      }
      if (Xc) {
        r(0 == K, 'cannot call main when async dependencies remain! (listen on Module["onRuntimeInitialized"])');
        r(0 == za.length, "cannot call main when preRun functions remain to be called");
        try {
          var b = Sc(0, 0);
          ac(b, !0);
        } catch (c) {
          cc(c);
        }
      }
      va();
      if (!p) {
        if (f.postRun) {
          for ("function" == typeof f.postRun && (f.postRun = [f.postRun]); f.postRun.length;) {
            b = f.postRun.shift(), Ca.unshift(b);
          }
        }
        Na(Ca);
      }
    }
  }
  if (!(0 < K)) {
    if (p || (r(!p), Tc(), ta()), p) {
      aa(f), Ga(), startWorker(f);
    } else {
      r(!p);
      if (f.preRun) {
        for ("function" == typeof f.preRun && (f.preRun = [f.preRun]); f.preRun.length;) {
          za.unshift(f.preRun.shift());
        }
      }
      Na(za);
      0 < K || (f.setStatus ? (f.setStatus("Running..."), setTimeout(function() {
        setTimeout(function() {
          f.setStatus("");
        }, 1);
        a();
      }, 1)) : a(), va());
    }
  }
}
function Zb() {
  var a = la, b = v, c = !1;
  la = v = () => {
    c = !0;
  };
  try {
    Rc(0), ["stdout", "stderr"].forEach(function(d) {
      d = "/dev/" + d;
      try {
        var e = T(d, {P:!0});
        d = e.path;
      } catch (k) {
      }
      var g = {Fa:!1, wa:!1, error:0, name:null, path:null, object:null, Ka:!1, Ma:null, La:null};
      try {
        e = T(d, {parent:!0}), g.Ka = !0, g.Ma = e.path, g.La = e.node, g.name = gb(d), e = T(d, {P:!0}), g.wa = !0, g.path = e.path, g.object = e.node, g.name = e.node.name, g.Fa = "/" === e.path;
      } catch (k) {
        g.error = k.F;
      }
      g && (e = pb[g.object.N]) && e.m && e.m.length && (c = !0);
    });
  } catch (d) {
  }
  la = a;
  v = b;
  c && P("stdio streams had content in them that was not flushed. you should set EXIT_RUNTIME to 1 (see the FAQ), or make sure to emit a newline when you printf etc.");
}
if (f.preInit) {
  for ("function" == typeof f.preInit && (f.preInit = [f.preInit]); 0 < f.preInit.length;) {
    f.preInit.pop()();
  }
}
var Xc = !0;
f.noInitialRun && (Xc = !1);
Wc();



  return moduleArg.ready
}

);
})();
export default MAGPIE;