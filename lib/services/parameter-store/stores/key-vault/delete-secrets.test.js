const mockBeginDeleteSecret = jest.fn(() =>
  Promise.resolve({ pollUntilDone: () => Promise.resolve() })
);
const mockPurgeDeletedSecret = jest.fn(() => Promise.resolve());

const { SecretClient } = require('@azure/keyvault-secrets');

SecretClient.mockImplementation(() => ({
  beginDeleteSecret: mockBeginDeleteSecret,
  purgeDeletedSecret: mockPurgeDeletedSecret
}));

const { makeKeyVaultStore } = require('./make-key-vault-store');

describe('deleteSecrets', () => {
  it('should delete and purge secrets', async () => {
    const kvStore = makeKeyVaultStore({ vaultName: 'test-vault' });

    await kvStore.deleteParameters({
      parameterNames: ['graph/DB_NAME', 'graph/DB_PASSWORD']
    });

    expect(mockBeginDeleteSecret).toBeCalledWith('graph--DB-NAME');
    expect(mockBeginDeleteSecret).toBeCalledWith('graph--DB-PASSWORD');
    expect(mockPurgeDeletedSecret).toBeCalledWith('graph--DB-NAME');
    expect(mockPurgeDeletedSecret).toBeCalledWith('graph--DB-PASSWORD');
  });
});
