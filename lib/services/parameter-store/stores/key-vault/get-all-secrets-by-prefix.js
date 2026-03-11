const { getKeyVaultClient } = require('./get-key-vault-client');
const { fromKeyVaultName, SEPARATOR } = require('./key-utils');

const getAllSecretsByPrefix = ({ vaultName }) => async ({ path }) => {
  const client = getKeyVaultClient({ vaultName });
  const prefix = `${path}${SEPARATOR}`;
  const secrets = [];

  for await (const properties of client.listPropertiesOfSecrets()) {
    if (properties.name.startsWith(prefix) && properties.enabled !== false) {
      const secret = await client.getSecret(properties.name);
      const tags = secret.properties.tags || {};

      secrets.push({
        Name: fromKeyVaultName(properties.name),
        Value: secret.value || '',
        Type: tags.type === 'config' ? 'String' : 'SecureString'
      });
    }
  }

  return secrets;
};

module.exports = { getAllSecretsByPrefix };
