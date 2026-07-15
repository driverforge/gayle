const { getKeyVaultClient } = require('./get-key-vault-client');
const { toKeyVaultName } = require('./key-utils');
const { logWarning } = require('../../../../utils/logger');

const deleteSecrets =
  ({ vaultName }) =>
  async ({ parameterNames }) => {
    const client = getKeyVaultClient({ vaultName });

    const deletions = parameterNames.map((name) => {
      const secretName = toKeyVaultName(name);
      return client
        .beginDeleteSecret(secretName)
        .then((poller) => poller.pollUntilDone())
        .then(() =>
          // Purge can legitimately fail (purge protection, RBAC): the
          // soft-deleted secret no longer appears in active listings, which
          // is what pruning needs — warn and move on. A failed *delete*, by
          // contrast, leaves the secret live and must reject loudly.
          client.purgeDeletedSecret(secretName).catch((error) => {
            logWarning(
              `Could not purge soft-deleted secret "${secretName}": ${error.message}`,
            );
          }),
        );
    });

    return Promise.all(deletions);
  };

module.exports = { deleteSecrets };
