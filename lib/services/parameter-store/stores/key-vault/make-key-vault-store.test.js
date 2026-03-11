const { makeKeyVaultStore } = require('./make-key-vault-store');

describe('makeKeyVaultStore', () => {
  it('should make an instance of key vault store', () => {
    const kvStore = makeKeyVaultStore({ vaultName: 'test-vault' });

    expect(kvStore).toHaveProperty('getAllParametersByNames');
    expect(kvStore).toHaveProperty('getAllParametersByPath');
    expect(kvStore).toHaveProperty('updateConfigs');
    expect(kvStore).toHaveProperty('updateSecrets');
    expect(kvStore).toHaveProperty('deleteParameters');
  });
});
