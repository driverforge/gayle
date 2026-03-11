const { getKeyVaultClient } = require('./get-key-vault-client');
const { toKeyVaultName } = require('./key-utils');

const deleteSecrets = ({ vaultName }) => async ({ parameterNames }) => {
  const client = getKeyVaultClient({ vaultName });

  const deletions = parameterNames.map(name => {
    const secretName = toKeyVaultName(name);
    return client
      .beginDeleteSecret(secretName)
      .then(poller => poller.pollUntilDone())
      .then(() => client.purgeDeletedSecret(secretName))
      .catch(() => {});
  });

  return Promise.all(deletions);
};

module.exports = { deleteSecrets };
