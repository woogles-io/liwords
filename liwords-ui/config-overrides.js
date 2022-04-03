const path = require('path');

module.exports = function override(config, env) {
  const wasmExtensionRegExp = /\.wasm$/;

  config.module.rules.forEach((rule) => {
    (rule.oneOf || []).forEach((oneOf) => {
      if (!oneOf.loader && oneOf.type === 'asset/resource') {
        oneOf.exclude.push(wasmExtensionRegExp);
      }
    });
  });

  config.experiments = {
    asyncWebAssembly: true,
  };

  return config;
};
