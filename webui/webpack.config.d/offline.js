// Production module contains no source maps or remotely loaded runtime resources.
config.plugins = config.plugins.filter(p => p.constructor.name !== 'SourceMapDevToolPlugin');
config.devtool = false;
// Webpack otherwise embeds Skiko's build-machine import.meta URL in the bundle.
// Keep the runtime base local and independent of the builder's absolute path.
class PortableSkikoURLPlugin {
  apply(compiler) {
    compiler.hooks.thisCompilation.tap('PortableSkikoURL', compilation => {
      compilation.hooks.processAssets.tap({
        name: 'PortableSkikoURL',
        stage: compiler.webpack.Compilation.PROCESS_ASSETS_STAGE_OPTIMIZE_INLINE
      }, assets => {
        for (const name of Object.keys(assets).filter(n => n.endsWith('.js'))) {
          const original = assets[name].source().toString();
          const portable = original.replace(/(["'])file:\/\/\/[^"'\r\n]+\/skiko\.mjs\1/g,
            'new URL("skiko.mjs",globalThis.location.href).href');
          if (portable !== original) compilation.updateAsset(name, new compiler.webpack.sources.RawSource(portable));
        }
      });
    });
  }
}
config.plugins.push(new PortableSkikoURLPlugin());
