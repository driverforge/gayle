const { makeGetAllSecretsByNames } = require('./make-get-all-secrets-by-names');

describe('makeGetAllSecretsByNames', () => {
  it('should return secrets keyed by their short name', () => {
    const loader = {
      loadMany: jest.fn(() => Promise.resolve(['value1', 'value2']))
    };

    const getAllSecretsByNames = makeGetAllSecretsByNames({ loader });

    return getAllSecretsByNames({
      parameterNames: ['graph/DB_NAME', 'graph/DB_HOST']
    }).then(result => {
      expect(result).toEqual({
        DB_NAME: 'value1',
        DB_HOST: 'value2'
      });
    });
  });
});
