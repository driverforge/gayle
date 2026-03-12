jest.mock('./stores/ssm/make-ssm-store', () => ({
  makeSsmStore: jest.fn(() => 'ssm'),
}));

jest.mock('./stores/key-vault/make-key-vault-store', () => ({
  makeKeyVaultStore: jest.fn(({ vaultName }) => `key-vault:${vaultName}`),
}));

const { makeGetProviderStore } = require('./make-get-provider-store');

describe('getProviderStore', () => {
  describe('when the provider is ssm', () => {
    it('should return the ssm store', () => {
      const getProviderStore = makeGetProviderStore({
        settingsService: {
          getSettings: () =>
            Promise.resolve({
              provider: {
                name: 'ssm',
              },
            }),
        },
      });

      expect.assertions(1);

      return getProviderStore().then((store) => {
        expect(store).toEqual('ssm');
      });
    });
  });

  describe('when the provider is key-vault', () => {
    it('should return the key vault store', () => {
      const getProviderStore = makeGetProviderStore({
        settingsService: {
          getSettings: () =>
            Promise.resolve({
              provider: {
                name: 'key-vault',
                vault: 'my-vault',
              },
            }),
        },
      });

      expect.assertions(1);

      return getProviderStore().then((store) => {
        expect(store).toEqual('key-vault:my-vault');
      });
    });
  });

  describe('when the provider is unsupported', () => {
    it('should throw an error', () => {
      const getProviderStore = makeGetProviderStore({
        settingsService: {
          getSettings: () =>
            Promise.resolve({
              provider: {
                name: 'somethingUnsupported',
              },
            }),
        },
      });

      expect.assertions(1);

      return expect(getProviderStore()).rejects.toEqual(
        new Error('Unsupported provider specified'),
      );
    });
  });
});
