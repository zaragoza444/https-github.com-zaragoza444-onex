const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const ts = require('typescript');

const sourcePath = path.join(__dirname, '..', 'src', 'plusConfig.ts');
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
  ALLTRA_PLUS_CHAIN_ID,
  ALLTRA_PLUS_EXPLORER_URL,
  ALLTRA_PLUS_RPC_URL,
  PLUS_ENDPOINTS,
  POUCHPAY_BASE_URL,
  POUCHPAY_PLUS_APP_NAME,
} = mod.exports;

assert.equal(POUCHPAY_PLUS_APP_NAME, 'PouchPay Plus');
assert.equal(POUCHPAY_BASE_URL, 'https://pouchpay.io/');
assert.equal(ALLTRA_PLUS_RPC_URL, 'https://mainnet-rpc.alltra.global/');
assert.equal(ALLTRA_PLUS_EXPLORER_URL, 'https://alltra.global/');
assert.equal(ALLTRA_PLUS_CHAIN_ID, 651940);
assert.deepEqual(
  PLUS_ENDPOINTS.map((endpoint) => endpoint.id),
  ['pouchpay', 'alltra-plus-rpc', 'alltra-plus-explorer'],
);

console.log('PouchPay Plus config tests passed');
