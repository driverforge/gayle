const mockGetSecret = jest.fn().mockImplementation(name => {
  const secrets = {
    'graph--DB-NAME': {
      value: 'my-db',
      properties: { tags: { type: 'config' } }
    },
    'graph--DB-HOST': {
      value: '3200',
      properties: { tags: { type: 'config' } }
    }
  };
  if (secrets[name]) {
    return Promise.resolve(secrets[name]);
  }
  return Promise.reject(new Error('Secret not found'));
});

const { SecretClient } = require('@azure/keyvault-secrets');

SecretClient.mockImplementation(() => ({
  getSecret: mockGetSecret
}));

const { getBatchSecrets } = require('./get-batch-secrets');

describe('getBatchSecrets', () => {
  let resultPromise;
  const batchFn = getBatchSecrets({ vaultName: 'test-vault' });

  beforeAll(() => {
    resultPromise = batchFn({
      parameterNames: ['graph/DB_NAME', 'graph/DB_HOST', 'graph/MISSING']
    });
    return resultPromise;
  });

  it('should fetch secrets from key vault', () => {
    expect(mockGetSecret).toBeCalledWith('graph--DB-NAME');
    expect(mockGetSecret).toBeCalledWith('graph--DB-HOST');
    expect(mockGetSecret).toBeCalledWith('graph--MISSING');
  });

  it('should return values in order and empty string for missing', () =>
    resultPromise.then(res => {
      expect(res).toEqual(['my-db', '3200', '']);
    }));
});
