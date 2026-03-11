jest.mock('./get-batch-secrets', () => ({
  getBatchSecrets: () => () =>
    Promise.resolve(['OLD_WORLD', 'OLD_BAR', 'UNCHANGED'])
}));

const mockSetSecret = jest.fn(() =>
  Promise.resolve({ properties: { version: '1' } })
);

const { SecretClient } = require('@azure/keyvault-secrets');

SecretClient.mockImplementation(() => ({
  setSecret: mockSetSecret
}));

const mockOnComplete = jest.fn();

const { makeKeyVaultStore } = require('./make-key-vault-store');

const kvStore = makeKeyVaultStore({ vaultName: 'test-vault' });

describe('updateSecrets', () => {
  describe('when an onCompleteHook is provided', () => {
    beforeAll(() =>
      kvStore.updateSecrets({
        parameters: {
          HELLO: 'WORLD',
          FOO: 'BAR',
          PARAM: 'UNCHANGED'
        },
        onComplete: mockOnComplete
      })
    );

    it('should update secrets in key vault with type secret tag', () => {
      expect(mockSetSecret).toBeCalledWith('--HELLO', 'WORLD', {
        tags: { type: 'secret' }
      });

      expect(mockSetSecret).toBeCalledWith('--FOO', 'BAR', {
        tags: { type: 'secret' }
      });
    });

    it('should not update secrets which have not changed', () => {
      expect(mockSetSecret).not.toBeCalledWith('--PARAM', 'UNCHANGED', {
        tags: { type: 'secret' }
      });

      expect(mockSetSecret.mock.calls.length).toEqual(2);
    });

    it('should run onComplete hook for each parameter', () => {
      expect(mockOnComplete).toBeCalledWith({
        name: 'HELLO',
        value: 'WORLD',
        version: '1'
      });

      expect(mockOnComplete).toBeCalledWith({
        name: 'FOO',
        value: 'BAR',
        version: '1'
      });
    });
  });

  describe('when an onComplete hook is not provided', () => {
    it('should still persist the secret in key vault', () => {
      mockSetSecret.mockClear();

      expect.assertions(1);

      return kvStore
        .updateSecrets({
          parameters: {
            HELLO: 'WORLD'
          }
        })
        .then(() => {
          expect(mockSetSecret).toBeCalledWith('--HELLO', 'WORLD', {
            tags: { type: 'secret' }
          });
        });
    });
  });
});
