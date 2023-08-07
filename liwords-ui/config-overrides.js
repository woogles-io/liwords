const path = require('path');

module.exports = {
  // reduce bundle size by only including antd icons we need
  resolve: {
    alias: {
      '@ant-design/icons$': path.resolve(__dirname, './src/icons.js'),
    },
  },

  webpack: function (config, env) {
    /*
    const wasmExtensionRegExp = /\.wasm$/;
    config.module.rules.forEach((rule) => {
      (rule.oneOf || []).forEach((oneOf) => {
        if (!oneOf.loader && oneOf.type === 'asset/resource') {
          oneOf.exclude.push(wasmExtensionRegExp);
        }
      });
    });
    */
    config.experiments = {
      asyncWebAssembly: true,
    };
    return config;
  },

  devServer: function (configFunction) {
    return function (proxy, allowedHost) {
      const config = configFunction(proxy, allowedHost);
      config.webSocketServer = config.webSocketServer || {};
      config.webSocketServer.options = {
        ...config.webSocketServer.options,
        path: process.env.WDS_SOCKET_PATH,
      };
      if (process.env.WDS_PROXY) {
        console.log('Webpack dev-server proxy is on...');
        config.proxy = {
          ...config.proxy,
          '/twirp': 'http://localhost:8001',
          '/gameimg': 'http://localhost:8001',
          '/ws': {
            target: 'ws://localhost:8087',
            ws: true,
          },
        };
      }
      config.client = {
        ...config.client,
        overlay: false,
      };
      return config;
    };
  },
};
