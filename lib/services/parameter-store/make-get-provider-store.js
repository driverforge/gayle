const get = require('lodash/get');
const { makeSsmStore } = require('./stores/ssm/make-ssm-store');
const {
  makeKeyVaultStore,
} = require('./stores/key-vault/make-key-vault-store');

const makeGetProviderStore =
  ({ settingsService }) =>
  () =>
    settingsService.getSettings().then((settings) => {
      const providerName = get(settings, 'provider.name');

      if (providerName === 'ssm') {
        return makeSsmStore({});
      }

      if (providerName === 'key-vault') {
        const vaultName = get(settings, 'provider.vault');
        return makeKeyVaultStore({ vaultName });
      }

      throw new Error('Unsupported provider specified');
    });

module.exports = {
  makeGetProviderStore,
};
