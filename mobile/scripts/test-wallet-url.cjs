const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const ts = require('typescript');

const sourcePath = path.join(__dirname, '..', 'src', 'walletUrl.ts');
const source = fs.readFileSync(sourcePath, 'utf8');
const { outputText } = ts.transpileModule(source, {
  compilerOptions: {
    module: ts.ModuleKind.CommonJS,
    target: ts.ScriptTarget.ES2022,
    strict: true,
  },
});

const mod = { exports: {} };
new Function('exports', 'require', 'module', '__filename', '__dirname', outputText)(
  mod.exports,
  require,
  mod,
  sourcePath,
  path.dirname(sourcePath),
);

const {
  normalizeWalletBaseUrl,
  shouldOpenWalletRequestExternally,
  walletUrlWithHash,
} = mod.exports;

const hostedWallet = 'https://git.anakatech.llc/pages/zaragoza/onex/wallet/';

assert.equal(
  normalizeWalletBaseUrl('https://example.com/wallet?bridge=https://bridge.example.com#swap'),
  'https://example.com/wallet/?bridge=https://bridge.example.com',
);
assert.equal(
  walletUrlWithHash('https://example.com/wallet?bridge=https://bridge.example.com', 'swap'),
  'https://example.com/wallet/?bridge=https://bridge.example.com#swap',
);
assert.equal(
  shouldOpenWalletRequestExternally(`${hostedWallet}#swap`, hostedWallet),
  false,
);
assert.equal(
  shouldOpenWalletRequestExternally(`${hostedWallet}assets/app.js`, hostedWallet),
  false,
);
assert.equal(
  shouldOpenWalletRequestExternally('https://git.anakatech.llc/bridge/status', hostedWallet),
  false,
);
assert.equal(
  shouldOpenWalletRequestExternally('https://git.anakatech.llc/pages/zaragoza/onex/wallet-v2/', hostedWallet),
  true,
);
assert.equal(
  shouldOpenWalletRequestExternally('https://example.com/wallet/', hostedWallet),
  true,
);
assert.equal(
  shouldOpenWalletRequestExternally('not a url', hostedWallet),
  true,
);

console.log('wallet URL policy tests passed');
