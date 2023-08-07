
var MAGPIE = (() => {
  var _scriptDir = import.meta.url;
  
  return (
async function(moduleArg = {}) {

var g = moduleArg, aa, ba;
g.ready = new Promise((a, b) => {
  aa = a;
  ba = b;
});
"_free _main _malloc _precache_file_data _process_ucgi_command _score_play __emscripten_thread_init __emscripten_thread_exit __emscripten_thread_crashed __emscripten_thread_mailbox_await __emscripten_tls_init _pthread_self checkMailbox establishStackSpace invokeEntryPoint PThread _fflush __emscripten_check_mailbox onRuntimeInitialized".split(" ").forEach(a => {
  Object.getOwnPropertyDescriptor(g.ready, a) || Object.defineProperty(g.ready, a, {get:() => l("You are getting " + a + " on the Promise object, instead of the instance. Use .then() to get called back with the instance, see the MODULARIZE docs in src/settings.js"), set:() => l("You are setting " + a + " on the Promise object, instead of the instance. Use .then() to get called back with the instance, see the MODULARIZE docs in src/settings.js"),});
});
var ca = Object.assign({}, g), da = (a, b) => {
  throw b;
}, ea = "object" == typeof window, n = "function" == typeof importScripts, p = "object" == typeof process && "object" == typeof process.versions && "string" == typeof process.versions.node, fa = !ea && !p && !n;
if (g.ENVIRONMENT) {
  throw Error("Module.ENVIRONMENT has been deprecated. To force the environment, use the ENVIRONMENT compile-time option (for example, -sENVIRONMENT=web or -sENVIRONMENT=node)");
}
var r = g.ENVIRONMENT_IS_PTHREAD || !1, u = "";
function ha(a) {
  return g.locateFile ? g.locateFile(a, u) : u + a;
}
var ia, ja, v;
if (p) {
  if ("undefined" == typeof process || !process.release || "node" !== process.release.name) {
    throw Error("not compiled for this environment (did you build to HTML and try to run it not on the web, or set ENVIRONMENT to something - like node - and run it someplace else - like on the web?)");
  }
  var ka = process.versions.node, la = ka.split(".").slice(0, 3);
  la = 10000 * la[0] + 100 * la[1] + 1 * la[2].split("-")[0];
  if (160000 > la) {
    throw Error("This emscripten-generated code requires node v16.0.0 (detected v" + ka + ")");
  }
  const {createRequire:a} = await import("module");
  var require = a(import.meta.url), fs = require("fs"), ma = require("path");
  n ? u = ma.dirname(u) + "/" : u = require("url").fileURLToPath(new URL("./", import.meta.url));
  ia = (c, d) => {
    c = na(c) ? new URL(c) : ma.normalize(c);
    return fs.readFileSync(c, d ? void 0 : "utf8");
  };
  v = c => {
    c = ia(c, !0);
    c.buffer || (c = new Uint8Array(c));
    assert(c.buffer);
    return c;
  };
  ja = (c, d, e, f = !0) => {
    c = na(c) ? new URL(c) : ma.normalize(c);
    fs.readFile(c, f ? void 0 : "utf8", (k, q) => {
      k ? e(k) : d(f ? q.buffer : q);
    });
  };
  process.argv.slice(2);
  da = (c, d) => {
    process.exitCode = c;
    throw d;
  };
  g.inspect = () => "[Emscripten Module object]";
  let b;
  try {
    b = require("worker_threads");
  } catch (c) {
    throw console.error('The "worker_threads" module is not supported in this node.js build - perhaps a newer version is needed?'), c;
  }
  global.Worker = b.Worker;
} else if (fa) {
  if ("object" == typeof process && "function" === typeof require || "object" == typeof window || "function" == typeof importScripts) {
    throw Error("not compiled for this environment (did you build to HTML and try to run it not on the web, or set ENVIRONMENT to something - like node - and run it someplace else - like on the web?)");
  }
  "undefined" != typeof read && (ia = a => read(a));
  v = a => {
    if ("function" == typeof readbuffer) {
      return new Uint8Array(readbuffer(a));
    }
    a = read(a, "binary");
    assert("object" == typeof a);
    return a;
  };
  ja = (a, b) => {
    setTimeout(() => b(v(a)));
  };
  "undefined" == typeof clearTimeout && (globalThis.clearTimeout = () => {
  });
  "undefined" == typeof setTimeout && (globalThis.setTimeout = a => "function" == typeof a ? a() : l());
  "function" == typeof quit && (da = (a, b) => {
    setTimeout(() => {
      if (!(b instanceof oa)) {
        let c = b;
        b && "object" == typeof b && b.stack && (c = [b, b.stack]);
        y(`exiting due to exception: ${c}`);
      }
      quit(a);
    });
    throw b;
  });
  "undefined" != typeof print && ("undefined" == typeof console && (console = {}), console.log = print, console.warn = console.error = "undefined" != typeof printErr ? printErr : print);
} else if (ea || n) {
  n ? u = self.location.href : "undefined" != typeof document && document.currentScript && (u = document.currentScript.src);
  _scriptDir && (u = _scriptDir);
  0 !== u.indexOf("blob:") ? u = u.substr(0, u.replace(/[?#].*/, "").lastIndexOf("/") + 1) : u = "";
  if ("object" != typeof window && "function" != typeof importScripts) {
    throw Error("not compiled for this environment (did you build to HTML and try to run it not on the web, or set ENVIRONMENT to something - like node - and run it someplace else - like on the web?)");
  }
  p || (ia = a => {
    var b = new XMLHttpRequest();
    b.open("GET", a, !1);
    b.send(null);
    return b.responseText;
  }, n && (v = a => {
    var b = new XMLHttpRequest();
    b.open("GET", a, !1);
    b.responseType = "arraybuffer";
    b.send(null);
    return new Uint8Array(b.response);
  }), ja = (a, b, c) => {
    var d = new XMLHttpRequest();
    d.open("GET", a, !0);
    d.responseType = "arraybuffer";
    d.onload = () => {
      200 == d.status || 0 == d.status && d.response ? b(d.response) : c();
    };
    d.onerror = c;
    d.send(null);
  });
} else {
  throw Error("environment detection error");
}
p && "undefined" == typeof performance && (global.performance = require("perf_hooks").performance);
var pa = console.log.bind(console), qa = console.error.bind(console);
p && (pa = (...a) => fs.writeSync(1, a.join(" ") + "\n"), qa = (...a) => fs.writeSync(2, a.join(" ") + "\n"));
var ra = g.print || pa, y = g.printErr || qa;
Object.assign(g, ca);
ca = null;
Object.getOwnPropertyDescriptor(g, "fetchSettings") && l("`Module.fetchSettings` was supplied but `fetchSettings` not included in INCOMING_MODULE_JS_API");
z("arguments", "arguments_");
z("thisProgram", "thisProgram");
g.quit && (da = g.quit);
z("quit", "quit_");
assert("undefined" == typeof g.memoryInitializerPrefixURL, "Module.memoryInitializerPrefixURL option was removed, use Module.locateFile instead");
assert("undefined" == typeof g.pthreadMainPrefixURL, "Module.pthreadMainPrefixURL option was removed, use Module.locateFile instead");
assert("undefined" == typeof g.cdInitializerPrefixURL, "Module.cdInitializerPrefixURL option was removed, use Module.locateFile instead");
assert("undefined" == typeof g.filePackagePrefixURL, "Module.filePackagePrefixURL option was removed, use Module.locateFile instead");
assert("undefined" == typeof g.read, "Module.read option was removed (modify read_ in JS)");
assert("undefined" == typeof g.readAsync, "Module.readAsync option was removed (modify readAsync in JS)");
assert("undefined" == typeof g.readBinary, "Module.readBinary option was removed (modify readBinary in JS)");
assert("undefined" == typeof g.setWindowTitle, "Module.setWindowTitle option was removed (modify setWindowTitle in JS)");
assert("undefined" == typeof g.TOTAL_MEMORY, "Module.TOTAL_MEMORY has been renamed Module.INITIAL_MEMORY");
z("read", "read_");
z("readAsync", "readAsync");
z("readBinary", "readBinary");
z("setWindowTitle", "setWindowTitle");
assert(ea || n || p, "Pthreads do not work in this environment yet (need Web Workers, or an alternative to them)");
assert(!fa, "shell environment detected but not enabled at build time.  Add 'shell' to `-sENVIRONMENT` to enable.");
var sa;
g.wasmBinary && (sa = g.wasmBinary);
z("wasmBinary", "wasmBinary");
var noExitRuntime = g.noExitRuntime || !0;
z("noExitRuntime", "noExitRuntime");
"object" != typeof WebAssembly && l("no native wasm support detected");
var B, ta, C = !1, ua;
function assert(a, b) {
  a || l("Assertion failed" + (b ? ": " + b : ""));
}
var E, va, wa, H, I, xa;
assert(!g.STACK_SIZE, "STACK_SIZE can no longer be set at runtime.  Use -sSTACK_SIZE at link time");
assert("undefined" != typeof Int32Array && "undefined" !== typeof Float64Array && void 0 != Int32Array.prototype.subarray && void 0 != Int32Array.prototype.set, "JS engine does not provide full typed array support");
var ya = g.INITIAL_MEMORY || 134217728;
z("INITIAL_MEMORY", "INITIAL_MEMORY");
assert(65536 <= ya, "INITIAL_MEMORY should be larger than STACK_SIZE, was " + ya + "! (STACK_SIZE=65536)");
if (r) {
  B = g.wasmMemory;
} else {
  if (g.wasmMemory) {
    B = g.wasmMemory;
  } else {
    if (B = new WebAssembly.Memory({initial:ya / 65536, maximum:ya / 65536, shared:!0}), !(B.buffer instanceof SharedArrayBuffer)) {
      throw y("requested a shared WebAssembly.Memory but the returned buffer is not a SharedArrayBuffer, indicating that while the browser has SharedArrayBuffer it does not have WebAssembly threads support - you may need to set a flag"), p && y("(on node you may need: --experimental-wasm-threads --experimental-wasm-bulk-memory and/or recent version)"), Error("bad memory");
    }
  }
}
var J = B.buffer;
g.HEAP8 = E = new Int8Array(J);
g.HEAP16 = wa = new Int16Array(J);
g.HEAP32 = H = new Int32Array(J);
g.HEAPU8 = va = new Uint8Array(J);
g.HEAPU16 = new Uint16Array(J);
g.HEAPU32 = I = new Uint32Array(J);
g.HEAPF32 = new Float32Array(J);
g.HEAPF64 = xa = new Float64Array(J);
ya = B.buffer.byteLength;
assert(0 === ya % 65536);
var za;
function Aa() {
  var a = Ba();
  assert(0 == (a & 3));
  0 == a && (a += 4);
  I[a >> 2] = 34821223;
  I[a + 4 >> 2] = 2310721022;
  I[0] = 1668509029;
}
function Ca() {
  if (!C) {
    var a = Ba();
    0 == a && (a += 4);
    var b = I[a >> 2], c = I[a + 4 >> 2];
    34821223 == b && 2310721022 == c || l(`Stack overflow! Stack cookie has been overwritten at ${Da(a)}, expected hex dwords 0x89BACDFE and 0x2135467, but received ${Da(c)} ${Da(b)}`);
    1668509029 != I[0] && l("Runtime error: The application has corrupted its heap memory area (address zero)!");
  }
}
var Ea = new Int16Array(1), Fa = new Int8Array(Ea.buffer);
Ea[0] = 25459;
if (115 !== Fa[0] || 99 !== Fa[1]) {
  throw "Runtime error: expected the system to be little-endian! (Run with -sSUPPORT_BIG_ENDIAN to bypass)";
}
var Ga = [], Ha = [], Ia = [], Ja = [], Ka = !1, La = 0;
function Ma() {
  return noExitRuntime || 0 < La;
}
function Na() {
  assert(!Ka);
  Ka = !0;
  if (!r) {
    Ca();
    if (!g.noFSInit && !Oa) {
      assert(!Oa, "FS.init was previously called. If you want to initialize later with custom parameters, remove any earlier calls (note that one is automatically added to the generated code)");
      Oa = !0;
      Pa();
      g.stdin = g.stdin;
      g.stdout = g.stdout;
      g.stderr = g.stderr;
      g.stdin ? Qa("stdin", g.stdin) : Ra("/dev/tty", "/dev/stdin");
      g.stdout ? Qa("stdout", null, g.stdout) : Ra("/dev/tty", "/dev/stdout");
      g.stderr ? Qa("stderr", null, g.stderr) : Ra("/dev/tty1", "/dev/stderr");
      var a = Sa("/dev/stdin", 0), b = Sa("/dev/stdout", 1), c = Sa("/dev/stderr", 1);
      assert(0 === a.fd, `invalid handle for stdin (${a.fd})`);
      assert(1 === b.fd, `invalid handle for stdout (${b.fd})`);
      assert(2 === c.fd, `invalid handle for stderr (${c.fd})`);
    }
    Ta = !1;
    Ua(Ha);
  }
}
assert(Math.imul, "This browser does not support Math.imul(), build with LEGACY_VM_SUPPORT or POLYFILL_OLD_MATH_FUNCTIONS to add in a polyfill");
assert(Math.fround, "This browser does not support Math.fround(), build with LEGACY_VM_SUPPORT or POLYFILL_OLD_MATH_FUNCTIONS to add in a polyfill");
assert(Math.clz32, "This browser does not support Math.clz32(), build with LEGACY_VM_SUPPORT or POLYFILL_OLD_MATH_FUNCTIONS to add in a polyfill");
assert(Math.trunc, "This browser does not support Math.trunc(), build with LEGACY_VM_SUPPORT or POLYFILL_OLD_MATH_FUNCTIONS to add in a polyfill");
var K = 0, L = null, Va = null, Wa = {};
function Xa(a) {
  K++;
  g.monitorRunDependencies && g.monitorRunDependencies(K);
  a ? (assert(!Wa[a]), Wa[a] = 1, null === L && "undefined" != typeof setInterval && (L = setInterval(() => {
    if (C) {
      clearInterval(L), L = null;
    } else {
      var b = !1, c;
      for (c in Wa) {
        b || (b = !0, y("still waiting on run dependencies:")), y("dependency: " + c);
      }
      b && y("(end of list)");
    }
  }, 10000))) : y("warning: run dependency added without ID");
}
function Ya(a) {
  K--;
  g.monitorRunDependencies && g.monitorRunDependencies(K);
  a ? (assert(Wa[a]), delete Wa[a]) : y("warning: run dependency removed without ID");
  0 == K && (null !== L && (clearInterval(L), L = null), Va && (a = Va, Va = null, a()));
}
function l(a) {
  if (g.onAbort) {
    g.onAbort(a);
  }
  a = "Aborted(" + a + ")";
  y(a);
  C = !0;
  ua = 1;
  a = new WebAssembly.RuntimeError(a);
  ba(a);
  throw a;
}
function Za(a) {
  return a.startsWith("data:application/octet-stream;base64,");
}
function na(a) {
  return a.startsWith("file://");
}
function N(a) {
  return function() {
    var b = g.asm;
    assert(Ka, "native function `" + a + "` called before runtime initialization");
    b[a] || assert(b[a], "exported native function `" + a + "` not found");
    return b[a].apply(null, arguments);
  };
}
var O;
g.locateFile ? (O = "magpie_wasm.wasm", Za(O) || (O = ha(O))) : O = (new URL("magpie_wasm.wasm", import.meta.url)).href;
function $a(a) {
  try {
    if (a == O && sa) {
      return new Uint8Array(sa);
    }
    if (v) {
      return v(a);
    }
    throw "both async and sync fetching of the wasm failed";
  } catch (b) {
    l(b);
  }
}
function ab(a) {
  if (!sa && (ea || n)) {
    if ("function" == typeof fetch && !na(a)) {
      return fetch(a, {credentials:"same-origin"}).then(b => {
        if (!b.ok) {
          throw "failed to load wasm binary file at '" + a + "'";
        }
        return b.arrayBuffer();
      }).catch(() => $a(a));
    }
    if (ja) {
      return new Promise((b, c) => {
        ja(a, d => b(new Uint8Array(d)), c);
      });
    }
  }
  return Promise.resolve().then(() => $a(a));
}
function bb(a, b, c) {
  return ab(a).then(d => WebAssembly.instantiate(d, b)).then(d => d).then(c, d => {
    y("failed to asynchronously prepare wasm: " + d);
    na(O) && y("warning: Loading from a file URI (" + O + ") is not supported in most browsers. See https://emscripten.org/docs/getting_started/FAQ.html#how-do-i-run-a-local-webserver-for-testing-why-does-my-program-stall-in-downloading-or-preparing");
    l(d);
  });
}
function cb(a, b) {
  var c = O;
  return sa || "function" != typeof WebAssembly.instantiateStreaming || Za(c) || na(c) || p || "function" != typeof fetch ? bb(c, a, b) : fetch(c, {credentials:"same-origin"}).then(d => WebAssembly.instantiateStreaming(d, a).then(b, function(e) {
    y("wasm streaming compile failed: " + e);
    y("falling back to ArrayBuffer instantiation");
    return bb(c, a, b);
  }));
}
var db, eb;
function z(a, b) {
  Object.getOwnPropertyDescriptor(g, a) || Object.defineProperty(g, a, {configurable:!0, get:function() {
    l("Module." + a + " has been replaced with plain " + b + " (the initial value can be provided on Module, but after startup the value is only looked for on a local variable of that name)");
  }});
}
function fb(a) {
  return "FS_createPath" === a || "FS_createDataFile" === a || "FS_createPreloadedFile" === a || "FS_unlink" === a || "addRunDependency" === a || "FS_createLazyFile" === a || "FS_createDevice" === a || "removeRunDependency" === a;
}
(function(a, b) {
  "undefined" !== typeof globalThis && Object.defineProperty(globalThis, a, {configurable:!0, get:function() {
    P("`" + a + "` is not longer defined by emscripten. " + b);
  }});
})("buffer", "Please use HEAP8.buffer or wasmMemory.buffer");
function gb(a) {
  Object.getOwnPropertyDescriptor(g, a) || Object.defineProperty(g, a, {configurable:!0, get:function() {
    var b = "'" + a + "' was not exported. add it to EXPORTED_RUNTIME_METHODS (see the FAQ)";
    fb(a) && (b += ". Alternatively, forcing filesystem support (-sFORCE_FILESYSTEM) can export this for you");
    l(b);
  }});
}
function oa(a) {
  this.name = "ExitStatus";
  this.message = `Program terminated with exit(${a})`;
  this.status = a;
}
function hb(a) {
  a.terminate();
  a.onmessage = b => {
    y('received "' + b.data.cmd + '" command from terminated worker: ' + a.ha);
  };
}
function ib(a) {
  assert(!r, "Internal Error! cleanupThread() can only ever be called from main application thread!");
  assert(a, "Internal Error! Null pthread_ptr in cleanupThread!");
  a = Q.o[a];
  assert(a);
  Q.Da(a);
}
function jb(a) {
  assert(!r, "Internal Error! spawnThread() can only ever be called from main application thread!");
  assert(a.l, "Internal error, no pthread ptr!");
  var b = Q.la();
  if (!b) {
    return 6;
  }
  assert(!b.l, "Internal error!");
  Q.G.push(b);
  Q.o[a.l] = b;
  b.l = a.l;
  var c = {cmd:"run", start_routine:a.Fa, arg:a.ia, pthread_ptr:a.l,};
  p && b.unref();
  b.postMessage(c, a.La);
  return 0;
}
var kb = (a, b) => {
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
}, lb = a => {
  var b = "/" === a.charAt(0), c = "/" === a.substr(-1);
  (a = kb(a.split("/").filter(d => !!d), !b).join("/")) || b || (a = ".");
  a && c && (a += "/");
  return (b ? "/" : "") + a;
}, mb = a => {
  var b = /^(\/?|)([\s\S]*?)((?:\.{1,2}|[^\/]+?|)(\.[^.\/]*|))(?:[\/]*)$/.exec(a).slice(1);
  a = b[0];
  b = b[1];
  if (!a && !b) {
    return ".";
  }
  b && (b = b.substr(0, b.length - 1));
  return a + b;
}, nb = a => {
  if ("/" === a) {
    return "/";
  }
  a = lb(a);
  a = a.replace(/\/$/, "");
  var b = a.lastIndexOf("/");
  return -1 === b ? a : a.substr(b + 1);
}, ob = () => {
  if ("object" == typeof crypto && "function" == typeof crypto.getRandomValues) {
    return c => (c.set(crypto.getRandomValues(new Uint8Array(c.byteLength))), c);
  }
  if (p) {
    try {
      var a = require("crypto");
      if (a.randomFillSync) {
        return c => a.randomFillSync(c);
      }
      var b = a.randomBytes;
      return c => (c.set(b(c.byteLength)), c);
    } catch (c) {
    }
  }
  l("no cryptographic support found for randomDevice. consider polyfilling it if you want to use something insecure like Math.random(), e.g. put this in a --pre-js: var crypto = { getRandomValues: (array) => { for (var i = 0; i < array.length; i++) array[i] = (Math.random()*256)|0 } };");
}, pb = a => (pb = ob())(a);
function qb() {
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
  a = kb(a.split("/").filter(d => !!d), !b).join("/");
  return (b ? "/" : "") + a || ".";
}
var rb = a => {
  for (var b = 0, c = 0; c < a.length; ++c) {
    var d = a.charCodeAt(c);
    127 >= d ? b++ : 2047 >= d ? b += 2 : 55296 <= d && 57343 >= d ? (b += 4, ++c) : b += 3;
  }
  return b;
}, sb = (a, b, c, d) => {
  assert("string" === typeof a);
  if (!(0 < d)) {
    return 0;
  }
  var e = c;
  d = c + d - 1;
  for (var f = 0; f < a.length; ++f) {
    var k = a.charCodeAt(f);
    if (55296 <= k && 57343 >= k) {
      var q = a.charCodeAt(++f);
      k = 65536 + ((k & 1023) << 10) | q & 1023;
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
          1114111 < k && P("Invalid Unicode code point " + Da(k) + " encountered when serializing a JS string to a UTF-8 string in wasm memory! (Valid unicode code points should be in range 0-0x10FFFF).");
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
function tb(a, b) {
  var c = Array(rb(a) + 1);
  a = sb(a, c, 0, c.length);
  b && (c.length = a);
  return c;
}
var ub = "undefined" != typeof TextDecoder ? new TextDecoder("utf8") : void 0, vb = (a, b) => {
  for (var c = b + NaN, d = b; a[d] && !(d >= c);) {
    ++d;
  }
  if (16 < d - b && a.buffer && ub) {
    return ub.decode(a.buffer instanceof SharedArrayBuffer ? a.slice(b, d) : a.subarray(b, d));
  }
  for (c = ""; b < d;) {
    var e = a[b++];
    if (e & 128) {
      var f = a[b++] & 63;
      if (192 == (e & 224)) {
        c += String.fromCharCode((e & 31) << 6 | f);
      } else {
        var k = a[b++] & 63;
        224 == (e & 240) ? e = (e & 15) << 12 | f << 6 | k : (240 != (e & 248) && P("Invalid UTF-8 leading byte " + Da(e) + " encountered when deserializing a UTF-8 string in wasm memory to a JS string!"), e = (e & 7) << 18 | f << 12 | k << 6 | a[b++] & 63);
        65536 > e ? c += String.fromCharCode(e) : (e -= 65536, c += String.fromCharCode(55296 | e >> 10, 56320 | e & 1023));
      }
    } else {
      c += String.fromCharCode(e);
    }
  }
  return c;
}, wb = [];
function xb(a, b) {
  wb[a] = {input:[], output:[], A:b};
  yb(a, zb);
}
var zb = {open:function(a) {
  var b = wb[a.node.rdev];
  if (!b) {
    throw new R(43);
  }
  a.tty = b;
  a.seekable = !1;
}, close:function(a) {
  a.tty.A.fsync(a.tty);
}, fsync:function(a) {
  a.tty.A.fsync(a.tty);
}, read:function(a, b, c, d) {
  if (!a.tty || !a.tty.A.Z) {
    throw new R(60);
  }
  for (var e = 0, f = 0; f < d; f++) {
    try {
      var k = a.tty.A.Z(a.tty);
    } catch (q) {
      throw new R(29);
    }
    if (void 0 === k && 0 === e) {
      throw new R(6);
    }
    if (null === k || void 0 === k) {
      break;
    }
    e++;
    b[c + f] = k;
  }
  e && (a.node.timestamp = Date.now());
  return e;
}, write:function(a, b, c, d) {
  if (!a.tty || !a.tty.A.R) {
    throw new R(60);
  }
  try {
    for (var e = 0; e < d; e++) {
      a.tty.A.R(a.tty, b[c + e]);
    }
  } catch (f) {
    throw new R(29);
  }
  d && (a.node.timestamp = Date.now());
  return e;
}}, Ab = {Z:function(a) {
  if (!a.input.length) {
    var b = null;
    if (p) {
      var c = Buffer.alloc(256), d = 0;
      try {
        d = fs.readSync(process.stdin.fd, c, 0, 256, -1);
      } catch (e) {
        if (e.toString().includes("EOF")) {
          d = 0;
        } else {
          throw e;
        }
      }
      0 < d ? b = c.slice(0, d).toString("utf-8") : b = null;
    } else {
      "undefined" != typeof window && "function" == typeof window.prompt ? (b = window.prompt("Input: "), null !== b && (b += "\n")) : "function" == typeof readline && (b = readline(), null !== b && (b += "\n"));
    }
    if (!b) {
      return null;
    }
    a.input = tb(b, !0);
  }
  return a.input.shift();
}, R:function(a, b) {
  null === b || 10 === b ? (ra(vb(a.output, 0)), a.output = []) : 0 != b && a.output.push(b);
}, fsync:function(a) {
  a.output && 0 < a.output.length && (ra(vb(a.output, 0)), a.output = []);
}, qa:function() {
  return {Ra:25856, Ta:5, Qa:191, Sa:35387, Pa:[3, 28, 127, 21, 4, 0, 1, 0, 17, 19, 26, 0, 18, 15, 23, 22, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,]};
}, ra:function() {
  return 0;
}, sa:function() {
  return [24, 80];
}}, Bb = {R:function(a, b) {
  null === b || 10 === b ? (y(vb(a.output, 0)), a.output = []) : 0 != b && a.output.push(b);
}, fsync:function(a) {
  a.output && 0 < a.output.length && (y(vb(a.output, 0)), a.output = []);
}}, S = {m:null, v:function() {
  return S.createNode(null, "/", 16895, 0);
}, createNode:function(a, b, c, d) {
  if (24576 === (c & 61440) || 4096 === (c & 61440)) {
    throw new R(63);
  }
  S.m || (S.m = {dir:{node:{C:S.h.C, s:S.h.s, lookup:S.h.lookup, L:S.h.L, rename:S.h.rename, unlink:S.h.unlink, rmdir:S.h.rmdir, readdir:S.h.readdir, symlink:S.h.symlink}, stream:{F:S.i.F}}, file:{node:{C:S.h.C, s:S.h.s}, stream:{F:S.i.F, read:S.i.read, write:S.i.write, U:S.i.U, aa:S.i.aa, da:S.i.da}}, link:{node:{C:S.h.C, s:S.h.s, readlink:S.h.readlink}, stream:{}}, W:{node:{C:S.h.C, s:S.h.s}, stream:Cb}});
  c = Db(a, b, c, d);
  16384 === (c.mode & 61440) ? (c.h = S.m.dir.node, c.i = S.m.dir.stream, c.g = {}) : 32768 === (c.mode & 61440) ? (c.h = S.m.file.node, c.i = S.m.file.stream, c.j = 0, c.g = null) : 40960 === (c.mode & 61440) ? (c.h = S.m.link.node, c.i = S.m.link.stream) : 8192 === (c.mode & 61440) && (c.h = S.m.W.node, c.i = S.m.W.stream);
  c.timestamp = Date.now();
  a && (a.g[b] = c, a.timestamp = c.timestamp);
  return c;
}, Wa:function(a) {
  return a.g ? a.g.subarray ? a.g.subarray(0, a.j) : new Uint8Array(a.g) : new Uint8Array(0);
}, X:function(a, b) {
  var c = a.g ? a.g.length : 0;
  c >= b || (b = Math.max(b, c * (1048576 > c ? 2.0 : 1.125) >>> 0), 0 != c && (b = Math.max(b, 256)), c = a.g, a.g = new Uint8Array(b), 0 < a.j && a.g.set(c.subarray(0, a.j), 0));
}, Ca:function(a, b) {
  if (a.j != b) {
    if (0 == b) {
      a.g = null, a.j = 0;
    } else {
      var c = a.g;
      a.g = new Uint8Array(b);
      c && a.g.set(c.subarray(0, Math.min(b, a.j)));
      a.j = b;
    }
  }
}, h:{C:function(a) {
  var b = {};
  b.dev = 8192 === (a.mode & 61440) ? a.id : 1;
  b.ino = a.id;
  b.mode = a.mode;
  b.nlink = 1;
  b.uid = 0;
  b.gid = 0;
  b.rdev = a.rdev;
  16384 === (a.mode & 61440) ? b.size = 4096 : 32768 === (a.mode & 61440) ? b.size = a.j : 40960 === (a.mode & 61440) ? b.size = a.link.length : b.size = 0;
  b.atime = new Date(a.timestamp);
  b.mtime = new Date(a.timestamp);
  b.ctime = new Date(a.timestamp);
  b.ja = 4096;
  b.blocks = Math.ceil(b.size / b.ja);
  return b;
}, s:function(a, b) {
  void 0 !== b.mode && (a.mode = b.mode);
  void 0 !== b.timestamp && (a.timestamp = b.timestamp);
  void 0 !== b.size && S.Ca(a, b.size);
}, lookup:function() {
  throw Eb[44];
}, L:function(a, b, c, d) {
  return S.createNode(a, b, c, d);
}, rename:function(a, b, c) {
  if (16384 === (a.mode & 61440)) {
    try {
      var d = Fb(b, c);
    } catch (f) {
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
}, unlink:function(a, b) {
  delete a.g[b];
  a.timestamp = Date.now();
}, rmdir:function(a, b) {
  var c = Fb(a, b), d;
  for (d in c.g) {
    throw new R(55);
  }
  delete a.g[b];
  a.timestamp = Date.now();
}, readdir:function(a) {
  var b = [".", ".."], c;
  for (c in a.g) {
    a.g.hasOwnProperty(c) && b.push(c);
  }
  return b;
}, symlink:function(a, b, c) {
  a = S.createNode(a, b, 41471, 0);
  a.link = c;
  return a;
}, readlink:function(a) {
  if (40960 !== (a.mode & 61440)) {
    throw new R(28);
  }
  return a.link;
}}, i:{read:function(a, b, c, d, e) {
  var f = a.node.g;
  if (e >= a.node.j) {
    return 0;
  }
  a = Math.min(a.node.j - e, d);
  assert(0 <= a);
  if (8 < a && f.subarray) {
    b.set(f.subarray(e, e + a), c);
  } else {
    for (d = 0; d < a; d++) {
      b[c + d] = f[e + d];
    }
  }
  return a;
}, write:function(a, b, c, d, e, f) {
  assert(!(b instanceof ArrayBuffer));
  if (!d) {
    return 0;
  }
  a = a.node;
  a.timestamp = Date.now();
  if (b.subarray && (!a.g || a.g.subarray)) {
    if (f) {
      return assert(0 === e, "canOwn must imply no weird position inside the file"), a.g = b.subarray(c, c + d), a.j = d;
    }
    if (0 === a.j && 0 === e) {
      return a.g = b.slice(c, c + d), a.j = d;
    }
    if (e + d <= a.j) {
      return a.g.set(b.subarray(c, c + d), e), d;
    }
  }
  S.X(a, e + d);
  if (a.g.subarray && b.subarray) {
    a.g.set(b.subarray(c, c + d), e);
  } else {
    for (f = 0; f < d; f++) {
      a.g[e + f] = b[c + f];
    }
  }
  a.j = Math.max(a.j, e + d);
  return d;
}, F:function(a, b, c) {
  1 === c ? b += a.position : 2 === c && 32768 === (a.node.mode & 61440) && (b += a.node.j);
  if (0 > b) {
    throw new R(28);
  }
  return b;
}, U:function(a, b, c) {
  S.X(a.node, b + c);
  a.node.j = Math.max(a.node.j, b + c);
}, aa:function(a, b, c, d, e) {
  if (32768 !== (a.node.mode & 61440)) {
    throw new R(43);
  }
  a = a.node.g;
  if (e & 2 || a.buffer !== E.buffer) {
    if (0 < c || c + b < a.length) {
      a.subarray ? a = a.subarray(c, c + b) : a = Array.prototype.slice.call(a, c, c + b);
    }
    c = !0;
    l("internal error: mmapAlloc called but `emscripten_builtin_memalign` native symbol not exported");
    b = void 0;
    if (!b) {
      throw new R(48);
    }
    E.set(a, b);
  } else {
    c = !1, b = a.byteOffset;
  }
  return {Ya:b, Oa:c};
}, da:function(a, b, c, d) {
  S.i.write(a, b, 0, d, c, !1);
  return 0;
}}};
function Gb(a, b) {
  var c = 0;
  a && (c |= 365);
  b && (c |= 146);
  return c;
}
var Hb = {0:"Success", 1:"Arg list too long", 2:"Permission denied", 3:"Address already in use", 4:"Address not available", 5:"Address family not supported by protocol family", 6:"No more processes", 7:"Socket already connected", 8:"Bad file number", 9:"Trying to read unreadable message", 10:"Mount device busy", 11:"Operation canceled", 12:"No children", 13:"Connection aborted", 14:"Connection refused", 15:"Connection reset by peer", 16:"File locking deadlock error", 17:"Destination address required", 
18:"Math arg out of domain of func", 19:"Quota exceeded", 20:"File exists", 21:"Bad address", 22:"File too large", 23:"Host is unreachable", 24:"Identifier removed", 25:"Illegal byte sequence", 26:"Connection already in progress", 27:"Interrupted system call", 28:"Invalid argument", 29:"I/O error", 30:"Socket is already connected", 31:"Is a directory", 32:"Too many symbolic links", 33:"Too many open files", 34:"Too many links", 35:"Message too long", 36:"Multihop attempted", 37:"File or path name too long", 
38:"Network interface is not configured", 39:"Connection reset by network", 40:"Network is unreachable", 41:"Too many open files in system", 42:"No buffer space available", 43:"No such device", 44:"No such file or directory", 45:"Exec format error", 46:"No record locks available", 47:"The link has been severed", 48:"Not enough core", 49:"No message of desired type", 50:"Protocol not available", 51:"No space left on device", 52:"Function not implemented", 53:"Socket is not connected", 54:"Not a directory", 
55:"Directory not empty", 56:"State not recoverable", 57:"Socket operation on non-socket", 59:"Not a typewriter", 60:"No such device or address", 61:"Value too large for defined data type", 62:"Previous owner died", 63:"Not super-user", 64:"Broken pipe", 65:"Protocol error", 66:"Unknown protocol", 67:"Protocol wrong type for socket", 68:"Math result not representable", 69:"Read only file system", 70:"Illegal seek", 71:"No such process", 72:"Stale file handle", 73:"Connection timed out", 74:"Text file busy", 
75:"Cross-device link", 100:"Device not a stream", 101:"Bad font file fmt", 102:"Invalid slot", 103:"Invalid request code", 104:"No anode", 105:"Block device required", 106:"Channel number out of range", 107:"Level 3 halted", 108:"Level 3 reset", 109:"Link number out of range", 110:"Protocol driver not attached", 111:"No CSI structure available", 112:"Level 2 halted", 113:"Invalid exchange", 114:"Invalid request descriptor", 115:"Exchange full", 116:"No data (for no delay io)", 117:"Timer expired", 
118:"Out of streams resources", 119:"Machine is not on the network", 120:"Package not installed", 121:"The object is remote", 122:"Advertise error", 123:"Srmount error", 124:"Communication error on send", 125:"Cross mount point (not really error)", 126:"Given log. name not unique", 127:"f.d. invalid for this operation", 128:"Remote address changed", 129:"Can   access a needed shared lib", 130:"Accessing a corrupted shared lib", 131:".lib section in a.out corrupted", 132:"Attempting to link in too many libs", 
133:"Attempting to exec a shared library", 135:"Streams pipe error", 136:"Too many users", 137:"Socket type not supported", 138:"Not supported", 139:"Protocol family not supported", 140:"Can't send after socket shutdown", 141:"Too many references", 142:"Host is down", 148:"No medium (in tape drive)", 156:"Level 2 not synchronized"}, Ib = {};
function Jb(a) {
  return a.replace(/\b_Z[\w\d_]+/g, function(b) {
    P("warning: build with -sDEMANGLE_SUPPORT to link in libcxxabi demangling");
    return b === b ? b : b + " [" + b + "]";
  });
}
var Kb = null, Lb = {}, Mb = [], Nb = 1, Ob = null, Ta = !0, R = null, Eb = {}, T = (a, b = {}) => {
  a = qb(a);
  if (!a) {
    return {path:"", node:null};
  }
  b = Object.assign({Y:!0, S:0}, b);
  if (8 < b.S) {
    throw new R(32);
  }
  a = a.split("/").filter(k => !!k);
  for (var c = Kb, d = "/", e = 0; e < a.length; e++) {
    var f = e === a.length - 1;
    if (f && b.parent) {
      break;
    }
    c = Fb(c, a[e]);
    d = lb(d + "/" + a[e]);
    c.M && (!f || f && b.Y) && (c = c.M.root);
    if (!f || b.J) {
      for (f = 0; 40960 === (c.mode & 61440);) {
        if (c = Pb(d), d = qb(mb(d), c), c = T(d, {S:b.S + 1}).node, 40 < f++) {
          throw new R(32);
        }
      }
    }
  }
  return {path:d, node:c};
}, Qb = a => {
  for (var b;;) {
    if (a === a.parent) {
      return a = a.v.ba, b ? "/" !== a[a.length - 1] ? `${a}/${b}` : a + b : a;
    }
    b = b ? `${a.name}/${b}` : a.name;
    a = a.parent;
  }
}, Rb = (a, b) => {
  for (var c = 0, d = 0; d < b.length; d++) {
    c = (c << 5) - c + b.charCodeAt(d) | 0;
  }
  return (a + c >>> 0) % Ob.length;
}, Fb = (a, b) => {
  var c;
  if (c = (c = Sb(a, "x")) ? c : a.h.lookup ? 0 : 2) {
    throw new R(c, a);
  }
  for (c = Ob[Rb(a.id, b)]; c; c = c.wa) {
    var d = c.name;
    if (c.parent.id === a.id && d === b) {
      return c;
    }
  }
  return a.h.lookup(a, b);
}, Db = (a, b, c, d) => {
  assert("object" == typeof a);
  a = new Tb(a, b, c, d);
  b = Rb(a.parent.id, a.name);
  a.wa = Ob[b];
  return Ob[b] = a;
}, Ub = a => {
  var b = ["r", "w", "rw"][a & 3];
  a & 512 && (b += "w");
  return b;
}, Sb = (a, b) => {
  if (Ta) {
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
}, Vb = (a, b) => {
  try {
    return Fb(a, b), 20;
  } catch (c) {
  }
  return Sb(a, "wx");
}, Wb = () => {
  for (var a = 0; 4096 >= a; a++) {
    if (!Mb[a]) {
      return a;
    }
  }
  throw new R(33);
}, U = a => {
  a = Mb[a];
  if (!a) {
    throw new R(8);
  }
  return a;
}, Yb = (a, b = -1) => {
  Xb || (Xb = function() {
    this.K = {};
  }, Xb.prototype = {}, Object.defineProperties(Xb.prototype, {object:{get:function() {
    return this.node;
  }, set:function(c) {
    this.node = c;
  }}, flags:{get:function() {
    return this.K.flags;
  }, set:function(c) {
    this.K.flags = c;
  },}, position:{get:function() {
    return this.K.position;
  }, set:function(c) {
    this.K.position = c;
  },},}));
  a = Object.assign(new Xb(), a);
  -1 == b && (b = Wb());
  a.fd = b;
  return Mb[b] = a;
}, Cb = {open:a => {
  a.i = Lb[a.node.rdev].i;
  a.i.open && a.i.open(a);
}, F:() => {
  throw new R(70);
}}, yb = (a, b) => {
  Lb[a] = {i:b};
}, Zb = (a, b) => {
  if ("string" == typeof a) {
    throw a;
  }
  var c = "/" === b, d = !b;
  if (c && Kb) {
    throw new R(10);
  }
  if (!c && !d) {
    var e = T(b, {Y:!1});
    b = e.path;
    e = e.node;
    if (e.M) {
      throw new R(10);
    }
    if (16384 !== (e.mode & 61440)) {
      throw new R(54);
    }
  }
  b = {type:a, Xa:{}, ba:b, va:[]};
  a = a.v(b);
  a.v = b;
  b.root = a;
  c ? Kb = a : e && (e.M = b, e.v && e.v.va.push(b));
}, V = (a, b, c) => {
  var d = T(a, {parent:!0}).node;
  a = nb(a);
  if (!a || "." === a || ".." === a) {
    throw new R(28);
  }
  var e = Vb(d, a);
  if (e) {
    throw new R(e);
  }
  if (!d.h.L) {
    throw new R(63);
  }
  return d.h.L(d, a, b, c);
}, $b = (a, b, c) => {
  "undefined" == typeof c && (c = b, b = 438);
  V(a, b | 8192, c);
}, Ra = (a, b) => {
  if (!qb(a)) {
    throw new R(44);
  }
  var c = T(b, {parent:!0}).node;
  if (!c) {
    throw new R(44);
  }
  b = nb(b);
  var d = Vb(c, b);
  if (d) {
    throw new R(d);
  }
  if (!c.h.symlink) {
    throw new R(63);
  }
  c.h.symlink(c, b, a);
}, Pb = a => {
  a = T(a).node;
  if (!a) {
    throw new R(44);
  }
  if (!a.h.readlink) {
    throw new R(28);
  }
  return qb(Qb(a.parent), a.h.readlink(a));
}, Sa = (a, b, c) => {
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
    a = lb(a);
    try {
      e = T(a, {J:!(b & 131072)}).node;
    } catch (f) {
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
  if (!d && (c = e ? 40960 === (e.mode & 61440) ? 32 : 16384 === (e.mode & 61440) && ("r" !== Ub(b) || b & 512) ? 31 : Sb(e, Ub(b)) : 44)) {
    throw new R(c);
  }
  if (b & 512 && !d) {
    c = e;
    c = "string" == typeof c ? T(c, {J:!0}).node : c;
    if (!c.h.s) {
      throw new R(63);
    }
    if (16384 === (c.mode & 61440)) {
      throw new R(31);
    }
    if (32768 !== (c.mode & 61440)) {
      throw new R(28);
    }
    if (d = Sb(c, "w")) {
      throw new R(d);
    }
    c.h.s(c, {size:0, timestamp:Date.now()});
  }
  b &= -131713;
  e = Yb({node:e, path:Qb(e), flags:b, seekable:!0, position:0, i:e.i, Ma:[], error:!1});
  e.i.open && e.i.open(e);
  !g.logReadFiles || b & 1 || (ac || (ac = {}), a in ac || (ac[a] = 1));
  return e;
}, bc = (a, b, c) => {
  if (null === a.fd) {
    throw new R(8);
  }
  if (!a.seekable || !a.i.F) {
    throw new R(70);
  }
  if (0 != c && 1 != c && 2 != c) {
    throw new R(28);
  }
  a.position = a.i.F(a, b, c);
  a.Ma = [];
}, Pa = () => {
  R || (R = function(a, b) {
    this.name = "ErrnoError";
    this.node = b;
    this.Ea = function(c) {
      this.B = c;
      for (var d in Ib) {
        if (Ib[d] === c) {
          this.code = d;
          break;
        }
      }
    };
    this.Ea(a);
    this.message = Hb[a];
    this.stack && (Object.defineProperty(this, "stack", {value:Error().stack, writable:!0}), this.stack = Jb(this.stack));
  }, R.prototype = Error(), R.prototype.constructor = R, [44].forEach(a => {
    Eb[a] = new R(a);
    Eb[a].stack = "<generic error, no stack>";
  }));
}, Oa, Qa = (a, b, c) => {
  a = lb("/dev/" + a);
  var d = Gb(!!b, !!c);
  cc || (cc = 64);
  var e = cc++ << 8 | 0;
  yb(e, {open:f => {
    f.seekable = !1;
  }, close:() => {
    c && c.buffer && c.buffer.length && c(10);
  }, read:(f, k, q, A) => {
    for (var m = 0, x = 0; x < A; x++) {
      try {
        var D = b();
      } catch (F) {
        throw new R(29);
      }
      if (void 0 === D && 0 === m) {
        throw new R(6);
      }
      if (null === D || void 0 === D) {
        break;
      }
      m++;
      k[q + x] = D;
    }
    m && (f.node.timestamp = Date.now());
    return m;
  }, write:(f, k, q, A) => {
    for (var m = 0; m < A; m++) {
      try {
        c(k[q + m]);
      } catch (x) {
        throw new R(29);
      }
    }
    A && (f.node.timestamp = Date.now());
    return m;
  }});
  $b(a, d, e);
}, cc, W = {}, Xb, ac, X = a => {
  assert("number" == typeof a);
  return a ? vb(va, a) : "";
}, dc = void 0;
function Y() {
  assert(void 0 != dc);
  dc += 4;
  return H[dc - 4 >> 2];
}
function ec(a) {
  if (r) {
    return Z(1, 1, a);
  }
  ua = a;
  if (!Ma()) {
    Q.Ga();
    if (g.onExit) {
      g.onExit(a);
    }
    C = !0;
  }
  da(a, new oa(a));
}
var hc = (a, b) => {
  ua = a;
  fc();
  if (r) {
    throw assert(!b), gc(a), "unwind";
  }
  Ma() && !b && (b = `program exited (with status: ${a}), but keepRuntimeAlive() is set (counter=${La}) due to an async operation, so halting execution but not exiting the runtime or preventing further async execution (you can use emscripten_force_exit, if you want to force a true shutdown)`, ba(b), y(b));
  ec(a);
}, Da = a => {
  assert("number" === typeof a);
  return "0x" + a.toString(16).padStart(8, "0");
}, jc = a => {
  a instanceof oa || "unwind" == a || (Ca(), a instanceof WebAssembly.RuntimeError && 0 >= ic() && y("Stack overflow detected.  You can try increasing -sSTACK_SIZE (currently set to 65536)"), da(1, a));
}, Q = {D:[], G:[], ga:[], o:{}, xa:1, Va:function() {
}, ma:function() {
  r ? Q.oa() : Q.na();
}, na:function() {
  for (var a = navigator.hardwareConcurrency + 3; a--;) {
    Q.V();
  }
  Ga.unshift(() => {
    Xa("loading-workers");
    Q.ua(() => Ya("loading-workers"));
  });
}, oa:function() {
  Q.receiveObjectTransfer = Q.Ba;
  Q.threadInitTLS = Q.fa;
  Q.setExitStatus = Q.ea;
  noExitRuntime = !1;
}, ea:function(a) {
  ua = a;
}, $a:["$terminateWorker"], Ga:function() {
  assert(!r, "Internal Error! terminateAllThreads() can only ever be called from main application thread!");
  for (var a of Q.G) {
    hb(a);
  }
  for (a of Q.D) {
    hb(a);
  }
  Q.D = [];
  Q.G = [];
  Q.o = [];
}, Da:function(a) {
  var b = a.l;
  delete Q.o[b];
  Q.D.push(a);
  Q.G.splice(Q.G.indexOf(a), 1);
  a.l = 0;
  kc(b);
}, Ba:function() {
}, fa:function() {
  Q.ga.forEach(a => a());
}, $:a => new Promise(b => {
  a.onmessage = f => {
    f = f.data;
    var k = f.cmd;
    a.l && (Q.ka = a.l);
    if (f.targetThread && f.targetThread != lc()) {
      var q = Q.o[f.Za];
      q ? q.postMessage(f, f.transferList) : y('Internal error! Worker sent a message "' + k + '" to target pthread ' + f.targetThread + ", but that thread no longer exists!");
    } else {
      if ("checkMailbox" === k) {
        mc();
      } else if ("spawnThread" === k) {
        jb(f);
      } else if ("cleanupThread" === k) {
        ib(f.thread);
      } else if ("killThread" === k) {
        f = f.thread, assert(!r, "Internal Error! killThread() can only ever be called from main application thread!"), assert(f, "Internal Error! Null pthread_ptr in killThread!"), k = Q.o[f], delete Q.o[f], hb(k), kc(f), Q.G.splice(Q.G.indexOf(k), 1), k.l = 0;
      } else if ("cancelThread" === k) {
        f = f.thread, assert(!r, "Internal Error! cancelThread() can only ever be called from main application thread!"), assert(f, "Internal Error! Null pthread_ptr in cancelThread!"), Q.o[f].postMessage({cmd:"cancel"});
      } else if ("loaded" === k) {
        a.loaded = !0, p && !a.l && a.unref(), b(a);
      } else if ("alert" === k) {
        alert("Thread " + f.threadId + ": " + f.text);
      } else if ("setimmediate" === f.target) {
        a.postMessage(f);
      } else if ("callHandler" === k) {
        g[f.handler](...f.args);
      } else {
        k && y("worker sent an unknown command " + k);
      }
    }
    Q.ka = void 0;
  };
  a.onerror = f => {
    var k = "worker sent an error!";
    a.l && (k = "Pthread " + Da(a.l) + " sent an error!");
    y(k + " " + f.filename + ":" + f.lineno + ": " + f.message);
    throw f;
  };
  p && (a.on("message", function(f) {
    a.onmessage({data:f});
  }), a.on("error", function(f) {
    a.onerror(f);
  }));
  assert(B instanceof WebAssembly.Memory, "WebAssembly memory should have been loaded by now!");
  assert(ta instanceof WebAssembly.Module, "WebAssembly Module should have been loaded by now!");
  var c = [], d = ["onExit", "onAbort", "print", "printErr",], e;
  for (e of d) {
    g.hasOwnProperty(e) && c.push(e);
  }
  a.ha = Q.xa++;
  a.postMessage({cmd:"load", handlers:c, urlOrBlob:g.mainScriptUrlOrBlob, wasmMemory:B, wasmModule:ta, workerID:a.ha,});
}), ua:function(a) {
  if (r) {
    return a();
  }
  Promise.all(Q.D.map(Q.$)).then(a);
}, V:function() {
  if (g.locateFile) {
    var a = ha("magpie_wasm.worker.js");
    a = new Worker(a);
  } else {
    a = new Worker(new URL("magpie_wasm.worker.js", import.meta.url));
  }
  Q.D.push(a);
}, la:function() {
  0 == Q.D.length && (p || y("Tried to spawn a new thread, but the thread pool is exhausted.\nThis might result in a deadlock unless some threads eventually exit or the code explicitly breaks out to the event loop.\nIf you want to increase the pool size, use setting `-sPTHREAD_POOL_SIZE=...`.\nIf you want to throw an explicit error instead of the risk of deadlocking in those cases, use setting `-sPTHREAD_POOL_SIZE_STRICT=2`."), Q.V(), Q.$(Q.D[0]));
  return Q.D.pop();
}};
g.PThread = Q;
var Ua = a => {
  for (; 0 < a.length;) {
    a.shift()(g);
  }
};
g.establishStackSpace = function() {
  var a = lc(), b = H[a + 52 >> 2];
  a = b - H[a + 56 >> 2];
  assert(0 != b);
  assert(0 != a);
  assert(b > a, "stackHigh must be higher then stackLow");
  nc(b, a);
  oc(b);
  Aa();
};
function gc(a) {
  if (r) {
    return Z(2, 0, a);
  }
  hc(a);
}
var pc = [];
g.invokeEntryPoint = function(a, b) {
  var c = pc[a];
  c || (a >= pc.length && (pc.length = a + 1), pc[a] = c = za.get(a));
  assert(za.get(a) == c, "JavaScript-side Wasm function table mirror is out of date!");
  a = c(b);
  Ca();
  Ma() ? Q.ea(a) : qc(a);
};
var P = a => {
  rc || (rc = {});
  rc[a] || (rc[a] = 1, p && (a = "warning: " + a), y(a));
}, rc;
function sc(a, b, c, d) {
  return r ? Z(3, 1, a, b, c, d) : tc(a, b, c, d);
}
function tc(a, b, c, d) {
  if ("undefined" == typeof SharedArrayBuffer) {
    return y("Current environment does not support SharedArrayBuffer, pthreads are not available!"), 6;
  }
  var e = [];
  if (r && 0 === e.length) {
    return sc(a, b, c, d);
  }
  a = {Fa:c, l:a, ia:d, La:e,};
  return r ? (a.Ua = "spawnThread", postMessage(a, e), 0) : jb(a);
}
function uc(a, b, c) {
  if (r) {
    return Z(4, 1, a, b, c);
  }
  dc = c;
  try {
    var d = U(a);
    switch(b) {
      case 0:
        var e = Y();
        return 0 > e ? -28 : Yb(d, e).fd;
      case 1:
      case 2:
        return 0;
      case 3:
        return d.flags;
      case 4:
        return e = Y(), d.flags |= e, 0;
      case 5:
        return e = Y(), wa[e + 0 >> 1] = 2, 0;
      case 6:
      case 7:
        return 0;
      case 16:
      case 8:
        return -28;
      case 9:
        return H[vc() >> 2] = 28, -1;
      default:
        return -28;
    }
  } catch (f) {
    if ("undefined" == typeof W || "ErrnoError" !== f.name) {
      throw f;
    }
    return -f.B;
  }
}
function wc(a, b, c) {
  if (r) {
    return Z(5, 1, a, b, c);
  }
  dc = c;
  try {
    var d = U(a);
    switch(b) {
      case 21509:
        return d.tty ? 0 : -59;
      case 21505:
        if (!d.tty) {
          return -59;
        }
        if (d.tty.A.qa) {
          b = [3, 28, 127, 21, 4, 0, 1, 0, 17, 19, 26, 0, 18, 15, 23, 22, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,];
          var e = Y();
          H[e >> 2] = 25856;
          H[e + 4 >> 2] = 5;
          H[e + 8 >> 2] = 191;
          H[e + 12 >> 2] = 35387;
          for (var f = 0; 32 > f; f++) {
            E[e + f + 17 >> 0] = b[f] || 0;
          }
        }
        return 0;
      case 21510:
      case 21511:
      case 21512:
        return d.tty ? 0 : -59;
      case 21506:
      case 21507:
      case 21508:
        if (!d.tty) {
          return -59;
        }
        if (d.tty.A.ra) {
          for (e = Y(), b = [], f = 0; 32 > f; f++) {
            b.push(E[e + f + 17 >> 0]);
          }
        }
        return 0;
      case 21519:
        if (!d.tty) {
          return -59;
        }
        e = Y();
        return H[e >> 2] = 0;
      case 21520:
        return d.tty ? -28 : -59;
      case 21531:
        e = Y();
        if (!d.i.pa) {
          throw new R(59);
        }
        return d.i.pa(d, b, e);
      case 21523:
        if (!d.tty) {
          return -59;
        }
        d.tty.A.sa && (f = [24, 80], e = Y(), wa[e >> 1] = f[0], wa[e + 2 >> 1] = f[1]);
        return 0;
      case 21524:
        return d.tty ? 0 : -59;
      case 21515:
        return d.tty ? 0 : -59;
      default:
        return -28;
    }
  } catch (k) {
    if ("undefined" == typeof W || "ErrnoError" !== k.name) {
      throw k;
    }
    return -k.B;
  }
}
function xc(a, b, c, d) {
  if (r) {
    return Z(6, 1, a, b, c, d);
  }
  dc = d;
  try {
    b = X(b);
    var e = b;
    if ("/" === e.charAt(0)) {
      b = e;
    } else {
      var f = -100 === a ? "/" : U(a).path;
      if (0 == e.length) {
        throw new R(44);
      }
      b = lb(f + "/" + e);
    }
    var k = d ? Y() : 0;
    return Sa(b, c, k).fd;
  } catch (q) {
    if ("undefined" == typeof W || "ErrnoError" !== q.name) {
      throw q;
    }
    return -q.B;
  }
}
var yc = a => {
  if (C) {
    y("user callback triggered after runtime exited or application aborted.  Ignoring.");
  } else {
    try {
      if (a(), !Ma()) {
        try {
          r ? qc(ua) : hc(ua);
        } catch (b) {
          jc(b);
        }
      }
    } catch (b) {
      jc(b);
    }
  }
};
function zc(a) {
  if ("function" === typeof Atomics.Na) {
    var b = Atomics.Na(H, a >> 2, a);
    assert(b.async);
    b.value.then(mc);
    Atomics.store(H, a + 128 >> 2, 1);
  }
}
g.__emscripten_thread_mailbox_await = zc;
function mc() {
  var a = lc();
  a && (zc(a), yc(() => Ac()));
}
g.checkMailbox = mc;
var Bc = a => 0 === a % 4 && (0 !== a % 100 || 0 === a % 400), Cc = [0, 31, 60, 91, 121, 152, 182, 213, 244, 274, 305, 335], Dc = [0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334], Ec = (a, b, c) => {
  assert("number" == typeof c, "stringToUTF8(str, outPtr, maxBytesToWrite) is missing the third parameter that specifies the length of the output buffer!");
  sb(a, va, b, c);
}, Gc = a => {
  var b = rb(a) + 1, c = Fc(b);
  c && Ec(a, c, b);
  return c;
}, Ic = a => {
  var b = Hc();
  a = a();
  oc(b);
  return a;
};
function Z(a, b) {
  var c = arguments.length - 2, d = arguments;
  if (19 < c) {
    throw "proxyToMainThread: Too many arguments " + c + " to proxied function idx=" + a + ", maximum supported is 19";
  }
  return Ic(() => {
    for (var e = Jc(8 * c), f = e >> 3, k = 0; k < c; k++) {
      xa[f + k] = d[2 + k];
    }
    return Kc(a, c, e, b);
  });
}
var Lc = [];
function Mc(a) {
  if (r) {
    return Z(7, 1, a);
  }
  try {
    var b = U(a);
    if (null === b.fd) {
      throw new R(8);
    }
    b.P && (b.P = null);
    try {
      b.i.close && b.i.close(b);
    } catch (c) {
      throw c;
    } finally {
      Mb[b.fd] = null;
    }
    b.fd = null;
    return 0;
  } catch (c) {
    if ("undefined" == typeof W || "ErrnoError" !== c.name) {
      throw c;
    }
    return c.B;
  }
}
function Nc(a, b, c, d) {
  if (r) {
    return Z(8, 1, a, b, c, d);
  }
  try {
    a: {
      var e = U(a);
      a = b;
      for (var f, k = b = 0; k < c; k++) {
        var q = I[a >> 2], A = I[a + 4 >> 2];
        a += 8;
        var m = e, x = q, D = A, F = f;
        if (0 > D || 0 > F) {
          throw new R(28);
        }
        if (null === m.fd) {
          throw new R(8);
        }
        if (1 === (m.flags & 2097155)) {
          throw new R(8);
        }
        if (16384 === (m.node.mode & 61440)) {
          throw new R(31);
        }
        if (!m.i.read) {
          throw new R(28);
        }
        var h = "undefined" != typeof F;
        if (!h) {
          F = m.position;
        } else if (!m.seekable) {
          throw new R(70);
        }
        var t = m.i.read(m, E, x, D, F);
        h || (m.position += t);
        var w = t;
        if (0 > w) {
          var G = -1;
          break a;
        }
        b += w;
        if (w < A) {
          break;
        }
        "undefined" !== typeof f && (f += w);
      }
      G = b;
    }
    I[d >> 2] = G;
    return 0;
  } catch (M) {
    if ("undefined" == typeof W || "ErrnoError" !== M.name) {
      throw M;
    }
    return M.B;
  }
}
function Oc(a, b, c, d, e) {
  if (r) {
    return Z(9, 1, a, b, c, d, e);
  }
  try {
    assert(b == b >>> 0 || b == (b | 0));
    assert(c === (c | 0));
    var f = c + 2097152 >>> 0 < 4194305 - !!b ? (b >>> 0) + 4294967296 * c : NaN;
    if (isNaN(f)) {
      return 61;
    }
    var k = U(a);
    bc(k, f, d);
    eb = [k.position >>> 0, (db = k.position, 1.0 <= +Math.abs(db) ? 0.0 < db ? +Math.floor(db / 4294967296.0) >>> 0 : ~~+Math.ceil((db - +(~~db >>> 0)) / 4294967296.0) >>> 0 : 0)];
    H[e >> 2] = eb[0];
    H[e + 4 >> 2] = eb[1];
    k.P && 0 === f && 0 === d && (k.P = null);
    return 0;
  } catch (q) {
    if ("undefined" == typeof W || "ErrnoError" !== q.name) {
      throw q;
    }
    return q.B;
  }
}
function Pc(a, b, c, d) {
  if (r) {
    return Z(10, 1, a, b, c, d);
  }
  try {
    a: {
      var e = U(a);
      a = b;
      for (var f, k = b = 0; k < c; k++) {
        var q = I[a >> 2], A = I[a + 4 >> 2];
        a += 8;
        var m = e, x = q, D = A, F = f;
        if (0 > D || 0 > F) {
          throw new R(28);
        }
        if (null === m.fd) {
          throw new R(8);
        }
        if (0 === (m.flags & 2097155)) {
          throw new R(8);
        }
        if (16384 === (m.node.mode & 61440)) {
          throw new R(31);
        }
        if (!m.i.write) {
          throw new R(28);
        }
        m.seekable && m.flags & 1024 && bc(m, 0, 2);
        var h = "undefined" != typeof F;
        if (!h) {
          F = m.position;
        } else if (!m.seekable) {
          throw new R(70);
        }
        var t = m.i.write(m, E, x, D, F, void 0);
        h || (m.position += t);
        var w = t;
        if (0 > w) {
          var G = -1;
          break a;
        }
        b += w;
        "undefined" !== typeof f && (f += w);
      }
      G = b;
    }
    I[d >> 2] = G;
    return 0;
  } catch (M) {
    if ("undefined" == typeof W || "ErrnoError" !== M.name) {
      throw M;
    }
    return M.B;
  }
}
var Qc = [31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31], Rc = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31], Sc = (a, b) => {
  assert(0 <= a.length, "writeArrayToMemory array must have a length (should be an array or typed array)");
  E.set(a, b);
};
function Tc(a) {
  var b = g["_" + a];
  assert(b, "Cannot call unknown function " + a + ", make sure it is exported");
  return b;
}
function Uc(a, b, c, d) {
  var e = {string:m => {
    var x = 0;
    if (null !== m && void 0 !== m && 0 !== m) {
      x = rb(m) + 1;
      var D = Jc(x);
      Ec(m, D, x);
      x = D;
    }
    return x;
  }, array:m => {
    var x = Jc(m.length);
    Sc(m, x);
    return x;
  }};
  a = Tc(a);
  var f = [], k = 0;
  assert("array" !== b, 'Return type should not be "array".');
  if (d) {
    for (var q = 0; q < d.length; q++) {
      var A = e[c[q]];
      A ? (0 === k && (k = Hc()), f[q] = A(d[q])) : f[q] = d[q];
    }
  }
  c = a.apply(null, f);
  return c = function(m) {
    0 !== k && oc(k);
    return "string" === b ? X(m) : "boolean" === b ? !!m : m;
  }(c);
}
Q.ma();
function Tb(a, b, c, d) {
  a || (a = this);
  this.parent = a;
  this.v = a.v;
  this.M = null;
  this.id = Nb++;
  this.name = b;
  this.mode = c;
  this.h = {};
  this.i = {};
  this.rdev = d;
}
Object.defineProperties(Tb.prototype, {read:{get:function() {
  return 365 === (this.mode & 365);
}, set:function(a) {
  a ? this.mode |= 365 : this.mode &= -366;
}}, write:{get:function() {
  return 146 === (this.mode & 146);
}, set:function(a) {
  a ? this.mode |= 146 : this.mode &= -147;
}}});
Pa();
Ob = Array(4096);
Zb(S, "/");
V("/tmp", 16895, 0);
V("/home", 16895, 0);
V("/home/web_user", 16895, 0);
(() => {
  V("/dev", 16895, 0);
  yb(259, {read:() => 0, write:(d, e, f, k) => k,});
  $b("/dev/null", 259);
  xb(1280, Ab);
  xb(1536, Bb);
  $b("/dev/tty", 1280);
  $b("/dev/tty1", 1536);
  var a = new Uint8Array(1024), b = 0, c = () => {
    0 === b && (b = pb(a).byteLength);
    return a[--b];
  };
  Qa("random", c);
  Qa("urandom", c);
  V("/dev/shm", 16895, 0);
  V("/dev/shm/tmp", 16895, 0);
})();
(() => {
  V("/proc", 16895, 0);
  var a = V("/proc/self", 16895, 0);
  V("/proc/self/fd", 16895, 0);
  Zb({v:() => {
    var b = Db(a, "fd", 16895, 73);
    b.h = {lookup:(c, d) => {
      var e = U(+d);
      c = {parent:null, v:{ba:"fake"}, h:{readlink:() => e.path},};
      return c.parent = c;
    }};
    return b;
  }}, "/proc/self/fd");
})();
Ib = {EPERM:63, ENOENT:44, ESRCH:71, EINTR:27, EIO:29, ENXIO:60, E2BIG:1, ENOEXEC:45, EBADF:8, ECHILD:12, EAGAIN:6, EWOULDBLOCK:6, ENOMEM:48, EACCES:2, EFAULT:21, ENOTBLK:105, EBUSY:10, EEXIST:20, EXDEV:75, ENODEV:43, ENOTDIR:54, EISDIR:31, EINVAL:28, ENFILE:41, EMFILE:33, ENOTTY:59, ETXTBSY:74, EFBIG:22, ENOSPC:51, ESPIPE:70, EROFS:69, EMLINK:34, EPIPE:64, EDOM:18, ERANGE:68, ENOMSG:49, EIDRM:24, ECHRNG:106, EL2NSYNC:156, EL3HLT:107, EL3RST:108, ELNRNG:109, EUNATCH:110, ENOCSI:111, EL2HLT:112, EDEADLK:16, 
ENOLCK:46, EBADE:113, EBADR:114, EXFULL:115, ENOANO:104, EBADRQC:103, EBADSLT:102, EDEADLOCK:16, EBFONT:101, ENOSTR:100, ENODATA:116, ETIME:117, ENOSR:118, ENONET:119, ENOPKG:120, EREMOTE:121, ENOLINK:47, EADV:122, ESRMNT:123, ECOMM:124, EPROTO:65, EMULTIHOP:36, EDOTDOT:125, EBADMSG:9, ENOTUNIQ:126, EBADFD:127, EREMCHG:128, ELIBACC:129, ELIBBAD:130, ELIBSCN:131, ELIBMAX:132, ELIBEXEC:133, ENOSYS:52, ENOTEMPTY:55, ENAMETOOLONG:37, ELOOP:32, EOPNOTSUPP:138, EPFNOSUPPORT:139, ECONNRESET:15, ENOBUFS:42, 
EAFNOSUPPORT:5, EPROTOTYPE:67, ENOTSOCK:57, ENOPROTOOPT:50, ESHUTDOWN:140, ECONNREFUSED:14, EADDRINUSE:3, ECONNABORTED:13, ENETUNREACH:40, ENETDOWN:38, ETIMEDOUT:73, EHOSTDOWN:142, EHOSTUNREACH:23, EINPROGRESS:26, EALREADY:7, EDESTADDRREQ:17, EMSGSIZE:35, EPROTONOSUPPORT:66, ESOCKTNOSUPPORT:137, EADDRNOTAVAIL:4, ENETRESET:39, EISCONN:30, ENOTCONN:53, ETOOMANYREFS:141, EUSERS:136, EDQUOT:19, ESTALE:72, ENOTSUP:138, ENOMEDIUM:148, EILSEQ:25, EOVERFLOW:61, ECANCELED:11, ENOTRECOVERABLE:56, EOWNERDEAD:62, 
ESTRPIPE:135,};
var Vc = [null, ec, gc, sc, uc, wc, xc, Mc, Nc, Oc, Pc], Xc = {__assert_fail:(a, b, c, d) => {
  l(`Assertion failed: ${X(a)}, at: ` + [b ? X(b) : "unknown filename", c, d ? X(d) : "unknown function"]);
}, __emscripten_init_main_thread_js:function(a) {
  Wc(a, !n, 1, !ea, 65536,);
  Q.fa();
}, __emscripten_thread_cleanup:function(a) {
  r ? postMessage({cmd:"cleanupThread", thread:a}) : ib(a);
}, __pthread_create_js:tc, __syscall_fcntl64:uc, __syscall_ioctl:wc, __syscall_openat:xc, _emscripten_get_now_is_monotonic:() => !0, _emscripten_notify_mailbox_postmessage:function(a, b) {
  a == b ? setTimeout(() => mc()) : r ? postMessage({targetThread:a, cmd:"checkMailbox"}) : (b = Q.o[a]) ? b.postMessage({cmd:"checkMailbox"}) : y("Cannot send message to thread with ID " + a + ", unknown thread ID!");
}, _emscripten_set_offscreencanvas_size:function() {
  y("emscripten_set_offscreencanvas_size: Build with -sOFFSCREENCANVAS_SUPPORT=1 to enable transferring canvases to pthreads.");
  return -1;
}, _emscripten_thread_mailbox_await:zc, _emscripten_thread_set_strongref:function(a) {
  p && Q.o[a].ref();
}, _localtime_js:(a, b) => {
  a = new Date(1000 * (I[a >> 2] + 4294967296 * H[a + 4 >> 2]));
  H[b >> 2] = a.getSeconds();
  H[b + 4 >> 2] = a.getMinutes();
  H[b + 8 >> 2] = a.getHours();
  H[b + 12 >> 2] = a.getDate();
  H[b + 16 >> 2] = a.getMonth();
  H[b + 20 >> 2] = a.getFullYear() - 1900;
  H[b + 24 >> 2] = a.getDay();
  H[b + 28 >> 2] = (Bc(a.getFullYear()) ? Cc : Dc)[a.getMonth()] + a.getDate() - 1 | 0;
  H[b + 36 >> 2] = -(60 * a.getTimezoneOffset());
  var c = (new Date(a.getFullYear(), 6, 1)).getTimezoneOffset(), d = (new Date(a.getFullYear(), 0, 1)).getTimezoneOffset();
  H[b + 32 >> 2] = (c != d && a.getTimezoneOffset() == Math.min(d, c)) | 0;
}, _tzset_js:(a, b, c) => {
  function d(A) {
    return (A = A.toTimeString().match(/\(([A-Za-z ]+)\)$/)) ? A[1] : "GMT";
  }
  var e = (new Date()).getFullYear(), f = new Date(e, 0, 1), k = new Date(e, 6, 1);
  e = f.getTimezoneOffset();
  var q = k.getTimezoneOffset();
  I[a >> 2] = 60 * Math.max(e, q);
  H[b >> 2] = Number(e != q);
  a = d(f);
  b = d(k);
  a = Gc(a);
  b = Gc(b);
  q < e ? (I[c >> 2] = a, I[c + 4 >> 2] = b) : (I[c >> 2] = b, I[c + 4 >> 2] = a);
}, abort:() => {
  l("native code called abort()");
}, emscripten_check_blocking_allowed:function() {
  p || n || (P("Blocking on the main thread is very dangerous, see https://emscripten.org/docs/porting/pthreads.html#blocking-on-the-main-browser-thread"), l("Blocking on the main thread is not allowed by default. See https://emscripten.org/docs/porting/pthreads.html#blocking-on-the-main-browser-thread"));
}, emscripten_date_now:function() {
  return Date.now();
}, emscripten_exit_with_live_runtime:() => {
  La += 1;
  throw "unwind";
}, emscripten_get_now:() => performance.timeOrigin + performance.now(), emscripten_receive_on_main_thread_js:function(a, b, c) {
  Lc.length = b;
  c >>= 3;
  for (var d = 0; d < b; d++) {
    Lc[d] = xa[c + d];
  }
  a = Vc[a];
  assert(a.length == b, "Call args mismatch in emscripten_receive_on_main_thread_js");
  return a.apply(null, Lc);
}, emscripten_resize_heap:a => {
  l(`Cannot enlarge memory arrays to size ${a >>> 0} bytes (OOM). Either (1) compile with -sINITIAL_MEMORY=X with X higher than the current value ${E.length}, (2) compile with -sALLOW_MEMORY_GROWTH which allows increasing the size at runtime, or (3) if you want malloc to return NULL (0) instead of this abort, compile with -sABORTING_MALLOC=0`);
}, exit:hc, fd_close:Mc, fd_read:Nc, fd_seek:Oc, fd_write:Pc, memory:B || g.wasmMemory, strftime:(a, b, c, d) => {
  function e(h, t, w) {
    for (h = "number" == typeof h ? h.toString() : h || ""; h.length < t;) {
      h = w[0] + h;
    }
    return h;
  }
  function f(h, t) {
    return e(h, t, "0");
  }
  function k(h, t) {
    function w(M) {
      return 0 > M ? -1 : 0 < M ? 1 : 0;
    }
    var G;
    0 === (G = w(h.getFullYear() - t.getFullYear())) && 0 === (G = w(h.getMonth() - t.getMonth())) && (G = w(h.getDate() - t.getDate()));
    return G;
  }
  function q(h) {
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
  function A(h) {
    var t = h.H;
    for (h = new Date((new Date(h.I + 1900, 0, 1)).getTime()); 0 < t;) {
      var w = h.getMonth(), G = (Bc(h.getFullYear()) ? Qc : Rc)[w];
      if (t > G - h.getDate()) {
        t -= G - h.getDate() + 1, h.setDate(1), 11 > w ? h.setMonth(w + 1) : (h.setMonth(0), h.setFullYear(h.getFullYear() + 1));
      } else {
        h.setDate(h.getDate() + t);
        break;
      }
    }
    w = new Date(h.getFullYear() + 1, 0, 4);
    t = q(new Date(h.getFullYear(), 0, 4));
    w = q(w);
    return 0 >= k(t, h) ? 0 >= k(w, h) ? h.getFullYear() + 1 : h.getFullYear() : h.getFullYear() - 1;
  }
  var m = H[d + 40 >> 2];
  d = {Ja:H[d >> 2], Ia:H[d + 4 >> 2], N:H[d + 8 >> 2], T:H[d + 12 >> 2], O:H[d + 16 >> 2], I:H[d + 20 >> 2], u:H[d + 24 >> 2], H:H[d + 28 >> 2], ab:H[d + 32 >> 2], Ha:H[d + 36 >> 2], Ka:m ? X(m) : ""};
  c = X(c);
  m = {"%c":"%a %b %d %H:%M:%S %Y", "%D":"%m/%d/%y", "%F":"%Y-%m-%d", "%h":"%b", "%r":"%I:%M:%S %p", "%R":"%H:%M", "%T":"%H:%M:%S", "%x":"%m/%d/%y", "%X":"%H:%M:%S", "%Ec":"%c", "%EC":"%C", "%Ex":"%m/%d/%y", "%EX":"%H:%M:%S", "%Ey":"%y", "%EY":"%Y", "%Od":"%d", "%Oe":"%e", "%OH":"%H", "%OI":"%I", "%Om":"%m", "%OM":"%M", "%OS":"%S", "%Ou":"%u", "%OU":"%U", "%OV":"%V", "%Ow":"%w", "%OW":"%W", "%Oy":"%y",};
  for (var x in m) {
    c = c.replace(new RegExp(x, "g"), m[x]);
  }
  var D = "Sunday Monday Tuesday Wednesday Thursday Friday Saturday".split(" "), F = "January February March April May June July August September October November December".split(" ");
  m = {"%a":h => D[h.u].substring(0, 3), "%A":h => D[h.u], "%b":h => F[h.O].substring(0, 3), "%B":h => F[h.O], "%C":h => f((h.I + 1900) / 100 | 0, 2), "%d":h => f(h.T, 2), "%e":h => e(h.T, 2, " "), "%g":h => A(h).toString().substring(2), "%G":h => A(h), "%H":h => f(h.N, 2), "%I":h => {
    h = h.N;
    0 == h ? h = 12 : 12 < h && (h -= 12);
    return f(h, 2);
  }, "%j":h => {
    for (var t = 0, w = 0; w <= h.O - 1; t += (Bc(h.I + 1900) ? Qc : Rc)[w++]) {
    }
    return f(h.T + t, 3);
  }, "%m":h => f(h.O + 1, 2), "%M":h => f(h.Ia, 2), "%n":() => "\n", "%p":h => 0 <= h.N && 12 > h.N ? "AM" : "PM", "%S":h => f(h.Ja, 2), "%t":() => "\t", "%u":h => h.u || 7, "%U":h => f(Math.floor((h.H + 7 - h.u) / 7), 2), "%V":h => {
    var t = Math.floor((h.H + 7 - (h.u + 6) % 7) / 7);
    2 >= (h.u + 371 - h.H - 2) % 7 && t++;
    if (t) {
      53 == t && (w = (h.u + 371 - h.H) % 7, 4 == w || 3 == w && Bc(h.I) || (t = 1));
    } else {
      t = 52;
      var w = (h.u + 7 - h.H - 1) % 7;
      (4 == w || 5 == w && Bc(h.I % 400 - 1)) && t++;
    }
    return f(t, 2);
  }, "%w":h => h.u, "%W":h => f(Math.floor((h.H + 7 - (h.u + 6) % 7) / 7), 2), "%y":h => (h.I + 1900).toString().substring(2), "%Y":h => h.I + 1900, "%z":h => {
    h = h.Ha;
    var t = 0 <= h;
    h = Math.abs(h) / 60;
    return (t ? "+" : "-") + String("0000" + (h / 60 * 100 + h % 60)).slice(-4);
  }, "%Z":h => h.Ka, "%%":() => "%"};
  c = c.replace(/%%/g, "\x00\x00");
  for (x in m) {
    c.includes(x) && (c = c.replace(new RegExp(x, "g"), m[x](d)));
  }
  c = c.replace(/\0\0/g, "%");
  x = tb(c, !1);
  if (x.length > b) {
    return 0;
  }
  Sc(x, a);
  return x.length - 1;
}};
(function() {
  function a(d, e) {
    d = d.exports;
    g.asm = d;
    Q.ga.push(g.asm._emscripten_tls_init);
    za = g.asm.__indirect_function_table;
    assert(za, "table not found in wasm exports");
    Ha.unshift(g.asm.__wasm_call_ctors);
    ta = e;
    Ya("wasm-instantiate");
    return d;
  }
  var b = {env:Xc, wasi_snapshot_preview1:Xc,};
  Xa("wasm-instantiate");
  var c = g;
  if (g.instantiateWasm) {
    try {
      return g.instantiateWasm(b, a);
    } catch (d) {
      y("Module.instantiateWasm callback failed with error: " + d), ba(d);
    }
  }
  cb(b, function(d) {
    assert(g === c, "the Module object should not be replaced during async compilation - perhaps the order of HTML elements is wrong?");
    c = null;
    a(d.instance, d.module);
  }).catch(ba);
  return {};
})();
var Fc = g._malloc = N("malloc");
g._free = N("free");
g._precache_file_data = N("precache_file_data");
var Yc = g._fflush = N("fflush");
g._process_ucgi_command = N("process_ucgi_command");
g._score_play = N("score_play");
var Zc = g._main = N("main");
g.__emscripten_tls_init = N("_emscripten_tls_init");
var lc = g._pthread_self = function() {
  return (lc = g._pthread_self = g.asm.pthread_self).apply(null, arguments);
}, vc = N("__errno_location"), Wc = g.__emscripten_thread_init = N("_emscripten_thread_init");
g.__emscripten_thread_crashed = N("_emscripten_thread_crashed");
var Kc = N("_emscripten_run_in_main_runtime_thread_js");
function Ba() {
  return (Ba = g.asm.emscripten_stack_get_end).apply(null, arguments);
}
var kc = N("_emscripten_thread_free_data"), qc = g.__emscripten_thread_exit = N("_emscripten_thread_exit"), Ac = g.__emscripten_check_mailbox = N("_emscripten_check_mailbox");
function $c() {
  return ($c = g.asm.emscripten_stack_init).apply(null, arguments);
}
function nc() {
  return (nc = g.asm.emscripten_stack_set_limits).apply(null, arguments);
}
var Hc = N("stackSave"), oc = N("stackRestore"), Jc = N("stackAlloc");
function ic() {
  return (ic = g.asm.emscripten_stack_get_current).apply(null, arguments);
}
g.dynCall_jiji = N("dynCall_jiji");
g.keepRuntimeAlive = Ma;
g.wasmMemory = B;
g.cwrap = function(a, b, c) {
  return function() {
    return Uc(a, b, c, arguments);
  };
};
g.stringToNewUTF8 = Gc;
g.ExitStatus = oa;
g.PThread = Q;
"growMemory inetPton4 inetNtop4 inetPton6 inetNtop6 readSockaddr writeSockaddr getHostByName traverseStack getCallstack emscriptenLog convertPCtoSourceLocation readEmAsmArgs jstoi_q jstoi_s getExecutableName listenOnce autoResumeAudioContext dynCallLegacy getDynCaller dynCall runtimeKeepalivePop safeSetTimeout asmjsMangle HandleAllocator getNativeTypeSize STACK_SIZE STACK_ALIGN POINTER_SIZE ASSERTIONS writeI53ToI64 writeI53ToI64Clamped writeI53ToI64Signaling writeI53ToU64Clamped writeI53ToU64Signaling readI53FromU64 convertI32PairToI53 convertU32PairToI53 uleb128Encode sigToWasmTypes generateFuncType convertJsFunctionToWasm getEmptyTableSlot updateTableMap getFunctionAddress addFunction removeFunction reallyNegative unSign strLen reSign formatString intArrayToString AsciiToString stringToAscii UTF16ToString stringToUTF16 lengthBytesUTF16 UTF32ToString stringToUTF32 lengthBytesUTF32 registerKeyEventCallback maybeCStringToJsString findEventTarget findCanvasEventTarget getBoundingClientRect fillMouseEventData registerMouseEventCallback registerWheelEventCallback registerUiEventCallback registerFocusEventCallback fillDeviceOrientationEventData registerDeviceOrientationEventCallback fillDeviceMotionEventData registerDeviceMotionEventCallback screenOrientation fillOrientationChangeEventData registerOrientationChangeEventCallback fillFullscreenChangeEventData registerFullscreenChangeEventCallback JSEvents_requestFullscreen JSEvents_resizeCanvasForFullscreen registerRestoreOldStyle hideEverythingExceptGivenElement restoreHiddenElements setLetterbox softFullscreenResizeWebGLRenderTarget doRequestFullscreen fillPointerlockChangeEventData registerPointerlockChangeEventCallback registerPointerlockErrorEventCallback requestPointerLock fillVisibilityChangeEventData registerVisibilityChangeEventCallback registerTouchEventCallback fillGamepadEventData registerGamepadEventCallback registerBeforeUnloadEventCallback fillBatteryEventData battery registerBatteryEventCallback setCanvasElementSizeCallingThread setCanvasElementSizeMainThread setCanvasElementSize getCanvasSizeCallingThread getCanvasSizeMainThread getCanvasElementSize jsStackTrace stackTrace getEnvStrings checkWasiClock wasiRightsToMuslOFlags wasiOFlagsToMuslOFlags createDyncallWrapper setImmediateWrapped clearImmediateWrapped polyfillSetImmediate getPromise makePromise idsToPromises makePromiseCallback ExceptionInfo setMainLoop getSocketFromFD getSocketAddress _setNetworkCallback heapObjectForWebGLType heapAccessShiftForWebGLHeap webgl_enable_ANGLE_instanced_arrays webgl_enable_OES_vertex_array_object webgl_enable_WEBGL_draw_buffers webgl_enable_WEBGL_multi_draw emscriptenWebGLGet computeUnpackAlignedImageSize colorChannelsInGlTextureFormat emscriptenWebGLGetTexPixelData __glGenObject emscriptenWebGLGetUniform webglGetUniformLocation webglPrepareUniformLocationsBeforeFirstUse webglGetLeftBracePos emscriptenWebGLGetVertexAttrib __glGetActiveAttribOrUniform writeGLArray emscripten_webgl_destroy_context_before_on_calling_thread registerWebGlEventCallback runAndAbortIfError SDL_unicode SDL_ttfContext SDL_audio GLFW_Window ALLOC_NORMAL ALLOC_STACK allocate writeStringToMemory writeAsciiToMemory".split(" ").forEach(function(a) {
  "undefined" === typeof globalThis || Object.getOwnPropertyDescriptor(globalThis, a) || Object.defineProperty(globalThis, a, {configurable:!0, get:function() {
    var b = "`" + a + "` is a library symbol and not included by default; add it to your library.js __deps or to DEFAULT_LIBRARY_FUNCS_TO_INCLUDE on the command line", c = a;
    c.startsWith("_") || (c = "$" + a);
    b += " (e.g. -sDEFAULT_LIBRARY_FUNCS_TO_INCLUDE='" + c + "')";
    fb(a) && (b += ". Alternatively, forcing filesystem support (-sFORCE_FILESYSTEM) can export this for you");
    P(b);
  }});
  gb(a);
});
"run addOnPreRun addOnInit addOnPreMain addOnExit addOnPostRun addRunDependency removeRunDependency FS_createFolder FS_createPath FS_createDataFile FS_createLazyFile FS_createLink FS_createDevice FS_unlink out err callMain abort stackAlloc stackSave stackRestore getTempRet0 setTempRet0 writeStackCookie checkStackCookie ptrToString zeroMemory exitJS getHeapMax abortOnCannotGrowMemory ENV MONTH_DAYS_REGULAR MONTH_DAYS_LEAP MONTH_DAYS_REGULAR_CUMULATIVE MONTH_DAYS_LEAP_CUMULATIVE isLeapYear ydayFromDate arraySum addDays ERRNO_CODES ERRNO_MESSAGES setErrNo DNS Protocols Sockets initRandomFill randomFill timers warnOnce UNWIND_CACHE readEmAsmArgsArray handleException runtimeKeepalivePush callUserCallback maybeExit asyncLoad alignMemory mmapAlloc readI53FromI64 convertI32PairToI53Checked getCFunc ccall freeTableIndexes functionsInTableMap setValue getValue PATH PATH_FS UTF8Decoder UTF8ArrayToString UTF8ToString stringToUTF8Array stringToUTF8 lengthBytesUTF8 intArrayFromString UTF16Decoder stringToUTF8OnStack writeArrayToMemory JSEvents specialHTMLTargets currentFullscreenStrategy restoreOldWindowedStyle demangle demangleAll doReadv doWritev promiseMap uncaughtExceptionCount exceptionLast exceptionCaught Browser wget SYSCALLS preloadPlugins FS_createPreloadedFile FS_modeStringToFlags FS_getMode FS MEMFS TTY PIPEFS SOCKFS tempFixedLengthArray miniTempWebGLFloatBuffers miniTempWebGLIntBuffers GL emscripten_webgl_power_preferences AL GLUT EGL GLEW IDBStore SDL SDL_gfx GLFW allocateUTF8 allocateUTF8OnStack terminateWorker killThread cleanupThread registerTLSInit cancelThread spawnThread exitOnMainThread proxyToMainThread emscripten_receive_on_main_thread_js_callArgs invokeEntryPoint checkMailbox".split(" ").forEach(gb);
var ad;
Va = function bd() {
  ad || cd();
  ad || (Va = bd);
};
function cd() {
  function a() {
    if (!ad && (ad = !0, g.calledRun = !0, !C)) {
      Na();
      Ca();
      r || Ua(Ia);
      aa(g);
      if (g.onRuntimeInitialized) {
        g.onRuntimeInitialized();
      }
      if (dd) {
        assert(0 == K, 'cannot call main when async dependencies remain! (listen on Module["onRuntimeInitialized"])');
        assert(0 == Ga.length, "cannot call main when preRun functions remain to be called");
        try {
          var b = Zc(0, 0);
          hc(b, !0);
        } catch (c) {
          jc(c);
        }
      }
      Ca();
      if (!r) {
        if (g.postRun) {
          for ("function" == typeof g.postRun && (g.postRun = [g.postRun]); g.postRun.length;) {
            b = g.postRun.shift(), Ja.unshift(b);
          }
        }
        Ua(Ja);
      }
    }
  }
  if (!(0 < K)) {
    if (r || (assert(!r), $c(), Aa()), r) {
      aa(g), Na(), startWorker(g);
    } else {
      assert(!r);
      if (g.preRun) {
        for ("function" == typeof g.preRun && (g.preRun = [g.preRun]); g.preRun.length;) {
          Ga.unshift(g.preRun.shift());
        }
      }
      Ua(Ga);
      0 < K || (g.setStatus ? (g.setStatus("Running..."), setTimeout(function() {
        setTimeout(function() {
          g.setStatus("");
        }, 1);
        a();
      }, 1)) : a(), Ca());
    }
  }
}
function fc() {
  var a = ra, b = y, c = !1;
  ra = y = () => {
    c = !0;
  };
  try {
    Yc(0), ["stdout", "stderr"].forEach(function(d) {
      d = "/dev/" + d;
      try {
        var e = T(d, {J:!0});
        d = e.path;
      } catch (k) {
      }
      var f = {ta:!1, exists:!1, error:0, name:null, path:null, object:null, ya:!1, Aa:null, za:null};
      try {
        e = T(d, {parent:!0}), f.ya = !0, f.Aa = e.path, f.za = e.node, f.name = nb(d), e = T(d, {J:!0}), f.exists = !0, f.path = e.path, f.object = e.node, f.name = e.node.name, f.ta = "/" === e.path;
      } catch (k) {
        f.error = k.B;
      }
      f && (e = wb[f.object.rdev]) && e.output && e.output.length && (c = !0);
    });
  } catch (d) {
  }
  ra = a;
  y = b;
  c && P("stdio streams had content in them that was not flushed. you should set EXIT_RUNTIME to 1 (see the FAQ), or make sure to emit a newline when you printf etc.");
}
if (g.preInit) {
  for ("function" == typeof g.preInit && (g.preInit = [g.preInit]); 0 < g.preInit.length;) {
    g.preInit.pop()();
  }
}
var dd = !0;
g.noInitialRun && (dd = !1);
cd();



  return moduleArg.ready
}

);
})();
export default MAGPIE;