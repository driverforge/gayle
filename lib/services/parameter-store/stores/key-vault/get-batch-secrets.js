const { getKeyVaultClient } = require('./get-key-vault-client');
const { toKeyVaultName } = require('./key-utils');

const getBatchSecrets = ({ vaultName }) => async ({ parameterNames }) => {
  const client = getKeyVaultClient({ vaultName });

  const results = await Promise.all(
    parameterNames.map(name => {
      const secretName = toKeyVaultName(name);
      return client
        .getSecret(secretName)
        .then(secret => secret.value || '')
        .catch(() => '');
    })
  );

  return results;
};

module.exports = { getBatchSecrets };
