const { getKeyVaultClient } = require('./get-key-vault-client');
const { toKeyVaultName } = require('./key-utils');

const makeUpdateConfig =
  ({ vaultName, getLatestVersion, onComplete }) =>
  async ({ key, value }) => {
    const client = getKeyVaultClient({ vaultName });
    const secretName = toKeyVaultName(key);

    const latestValue = await getLatestVersion(key);

    if (latestValue === value) {
      return Promise.resolve();
    }

    const result = await client.setSecret(secretName, value, {
      tags: { type: 'config' },
    });

    return onComplete({
      name: key,
      value,
      version: result.properties.version,
    });
  };

const makeUpdateConfigs =
  ({ vaultName, getAllSecretsByNames, getLatestVersion }) =>
  async ({ parameters, onComplete = () => Promise.resolve() }) => {
    const parameterNames = Object.keys(parameters);
    await getAllSecretsByNames({ parameterNames });

    const updateConfig = makeUpdateConfig({
      vaultName,
      getLatestVersion,
      onComplete,
    });

    const updaters = Object.entries(parameters).map(
      ([key, value]) =>
        () =>
          updateConfig({ key, value }),
    );

    return Promise.all(updaters.map((updater) => updater()));
  };

module.exports = { makeUpdateConfigs };
