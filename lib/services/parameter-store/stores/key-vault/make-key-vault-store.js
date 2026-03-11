const DataLoader = require('dataloader');
const { makeUpdateConfigs } = require('./make-update-configs');
const { makeUpdateSecrets } = require('./make-update-secrets');
const { getBatchSecrets } = require('./get-batch-secrets');
const {
  makeGetAllSecretsByNames
} = require('./make-get-all-secrets-by-names');
const { getAllSecretsByPrefix } = require('./get-all-secrets-by-prefix');
const { deleteSecrets } = require('./delete-secrets');

const makeKeyVaultStore = ({ vaultName }) => {
  const batchFn = getBatchSecrets({ vaultName });
  const kvLoader = new DataLoader(keys => batchFn({ parameterNames: keys }));
  const getAllSecretsByNames = makeGetAllSecretsByNames({ loader: kvLoader });
  const getLatestVersion = key => kvLoader.load(key);

  return {
    getAllParametersByPath: getAllSecretsByPrefix({ vaultName }),
    getAllParametersByNames: getAllSecretsByNames,
    updateConfigs: makeUpdateConfigs({
      vaultName,
      getAllSecretsByNames,
      getLatestVersion
    }),
    updateSecrets: makeUpdateSecrets({
      vaultName,
      getAllSecretsByNames,
      getLatestVersion
    }),
    deleteParameters: deleteSecrets({ vaultName })
  };
};

module.exports = { makeKeyVaultStore };
