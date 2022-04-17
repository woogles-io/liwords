const path = require('path');

module.exports = {
  webpack: function(config, env) {
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
  },

  devServer: function(configFunction) {
    return function(proxy, allowedHost) {
      const config = configFunction(proxy, allowedHost);
      config.webSocketServer = config.webSocketServer || {};
      config.webSocketServer.options = {
        ...config.webSocketServer.options,
        path: process.env.WDS_SOCKET_PATH,
      };
      config.client = {
        ...config.client,
        overlay: false,
      };
      return config;
    }
  }
};
