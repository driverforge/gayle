const mockBeginDeleteSecret = jest.fn(() =>
  Promise.resolve({ pollUntilDone: () => Promise.resolve() }),
);
const mockPurgeDeletedSecret = jest.fn(() => Promise.resolve());

const { SecretClient } = require('@azure/keyvault-secrets');

SecretClient.mockImplementation(() => ({
  beginDeleteSecret: mockBeginDeleteSecret,
  purgeDeletedSecret: mockPurgeDeletedSecret,
}));

const mockLogWarning = jest.fn();

jest.mock('../../../../utils/logger', () => ({
  log: jest.fn(),
  logError: jest.fn(),
  logWarning: mockLogWarning,
}));

const { makeKeyVaultStore } = require('./make-key-vault-store');

describe('deleteSecrets', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockBeginDeleteSecret.mockImplementation(() =>
      Promise.resolve({ pollUntilDone: () => Promise.resolve() }),
    );
    mockPurgeDeletedSecret.mockImplementation(() => Promise.resolve());
  });

  it('should delete and purge secrets', async () => {
    const kvStore = makeKeyVaultStore({ vaultName: 'test-vault' });

    await kvStore.deleteParameters({
      parameterNames: ['graph/DB_NAME', 'graph/DB_PASSWORD'],
    });

    expect(mockBeginDeleteSecret).toHaveBeenCalledWith('graph--DB-NAME');
    expect(mockBeginDeleteSecret).toHaveBeenCalledWith('graph--DB-PASSWORD');
    expect(mockPurgeDeletedSecret).toHaveBeenCalledWith('graph--DB-NAME');
    expect(mockPurgeDeletedSecret).toHaveBeenCalledWith('graph--DB-PASSWORD');
  });

  it('should reject when a deletion fails, leaving the secret live', async () => {
    mockBeginDeleteSecret.mockImplementationOnce(() =>
      Promise.reject(new Error('forbidden')),
    );
    const kvStore = makeKeyVaultStore({ vaultName: 'test-vault' });

    await expect(
      kvStore.deleteParameters({ parameterNames: ['graph/DB_NAME'] }),
    ).rejects.toThrow('forbidden');
  });

  it('should warn but resolve when only the purge fails', async () => {
    mockPurgeDeletedSecret.mockImplementationOnce(() =>
      Promise.reject(new Error('purge protection enabled')),
    );
    const kvStore = makeKeyVaultStore({ vaultName: 'test-vault' });

    await expect(
      kvStore.deleteParameters({ parameterNames: ['graph/DB_NAME'] }),
    ).resolves.toBeDefined();
    expect(mockLogWarning).toHaveBeenCalledWith(
      expect.stringContaining('graph--DB-NAME'),
    );
  });
});
