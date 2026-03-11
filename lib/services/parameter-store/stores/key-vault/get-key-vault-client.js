const { SecretClient } = require('@azure/keyvault-secrets');
const { DefaultAzureCredential } = require('@azure/identity');

const clients = {};

const getKeyVaultClient = ({ vaultName }) => {
  if (!clients[vaultName]) {
    const vaultUrl = `https://${vaultName}.vault.azure.net`;
    clients[vaultName] = new SecretClient(vaultUrl, new DefaultAzureCredential());
  }
  return clients[vaultName];
};

module.exports = { getKeyVaultClient };
