const SecretClient = jest.fn(() => ({
  getSecret: jest.fn(() => Promise.resolve({ value: '', properties: { tags: {} } })),
  setSecret: jest.fn(() =>
    Promise.resolve({ properties: { version: '1' } })
  ),
  beginDeleteSecret: jest.fn(() =>
    Promise.resolve({ pollUntilDone: () => Promise.resolve() })
  ),
  purgeDeletedSecret: jest.fn(() => Promise.resolve()),
  listPropertiesOfSecrets: jest.fn(() => [])
}));

module.exports = { SecretClient };
