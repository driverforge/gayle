const mockGetSecret = jest.fn((name) => {
  const secrets = {
    'graph--DB-NAME': {
      value: 'my-db',
      properties: { name: 'graph--DB-NAME', tags: { type: 'config' } },
    },
    'graph--DB-PASSWORD': {
      value: 's3cret',
      properties: { name: 'graph--DB-PASSWORD', tags: { type: 'secret' } },
    },
  };
  return Promise.resolve(secrets[name]);
});

const mockListProperties = [
  { name: 'graph--DB-NAME', enabled: true },
  { name: 'graph--DB-PASSWORD', enabled: true },
  { name: 'other--API-KEY', enabled: true },
];

const { SecretClient } = require('@azure/keyvault-secrets');

SecretClient.mockImplementation(() => ({
  getSecret: mockGetSecret,
  listPropertiesOfSecrets: jest.fn(() => mockListProperties),
}));

const { getAllSecretsByPrefix } = require('./get-all-secrets-by-prefix');

describe('getAllSecretsByPrefix', () => {
  it('should return only secrets matching the prefix', async () => {
    const fn = getAllSecretsByPrefix({ vaultName: 'test-vault' });
    const results = await fn({ path: 'graph' });

    expect(results).toEqual([
      { Name: 'graph/DB_NAME', Value: 'my-db', Type: 'String' },
      { Name: 'graph/DB_PASSWORD', Value: 's3cret', Type: 'SecureString' },
    ]);
  });

  it('should not fetch secrets that do not match the prefix', async () => {
    expect(mockGetSecret).not.toHaveBeenCalledWith('other--API-KEY');
  });
});
