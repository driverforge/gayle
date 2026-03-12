jest.mock('./get-batch-secrets', () => ({
  getBatchSecrets: () => () =>
    Promise.resolve(['OLD_WORLD', 'OLD_BAR', 'UNCHANGED']),
}));

const mockSetSecret = jest.fn(() =>
  Promise.resolve({ properties: { version: '1' } }),
);

const { SecretClient } = require('@azure/keyvault-secrets');

SecretClient.mockImplementation(() => ({
  setSecret: mockSetSecret,
}));

const mockOnComplete = jest.fn();

const { makeKeyVaultStore } = require('./make-key-vault-store');

const kvStore = makeKeyVaultStore({ vaultName: 'test-vault' });

describe('updateConfigs', () => {
  describe('when an onCompleteHook is provided', () => {
    beforeAll(() =>
      kvStore.updateConfigs({
        parameters: {
          HELLO: 'WORLD',
          FOO: 'BAR',
          GREAT: 'UNCHANGED',
        },
        onComplete: mockOnComplete,
      }),
    );

    it('should update configs in key vault with type config tag', () => {
      expect(mockSetSecret).toHaveBeenCalledWith('--HELLO', 'WORLD', {
        tags: { type: 'config' },
      });

      expect(mockSetSecret).toHaveBeenCalledWith('--FOO', 'BAR', {
        tags: { type: 'config' },
      });
    });

    it('should not update configs which have not changed', () => {
      expect(mockSetSecret).not.toHaveBeenCalledWith('--GREAT', 'UNCHANGED', {
        tags: { type: 'config' },
      });

      expect(mockSetSecret.mock.calls.length).toEqual(2);
    });

    it('should run onComplete hook for each parameter', () => {
      expect(mockOnComplete).toHaveBeenCalledWith({
        name: 'HELLO',
        value: 'WORLD',
        version: '1',
      });

      expect(mockOnComplete).toHaveBeenCalledWith({
        name: 'FOO',
        value: 'BAR',
        version: '1',
      });
    });
  });

  describe('when an onComplete hook is not provided', () => {
    it('should still persist the config in key vault', () => {
      mockSetSecret.mockClear();

      expect.assertions(1);

      return kvStore
        .updateConfigs({
          parameters: {
            HELLO: 'WORLD',
          },
        })
        .then(() => {
          expect(mockSetSecret).toHaveBeenCalledWith('--HELLO', 'WORLD', {
            tags: { type: 'config' },
          });
        });
    });
  });
});
