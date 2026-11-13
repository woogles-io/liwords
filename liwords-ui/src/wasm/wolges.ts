// The wasm, instantiated: the web build fetches its .wasm on the first call
// and answers every later call with the same instance.
export const importWolges = () =>
  import("wolges-wasm").then(async (wolges) => {
    await wolges.default();
    return wolges;
  });
