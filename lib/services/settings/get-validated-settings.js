const get = require('lodash/get');

const ALLOWED_PROVIDERS = ['ssm', 'ddb', 'key-vault'];

const isAllowedProvider = config => {
  const providerName = get(config, 'provider.name');

  if (!ALLOWED_PROVIDERS.includes(providerName)) {
    throw new Error(
      `Invalid provider '${providerName}'!! Only ${ALLOWED_PROVIDERS} are supported.`
    );
  }
};

const validateProvider = config => {
  const provider = get(config, 'provider');
  const name = get(provider, 'name');

  if (name === 'ddb') {
    const tableName = get(provider, 'tableName');

    if (!tableName) {
      throw new Error(
        `Invalid provider!! 'provider.tableName' must be passed for 'ddb' provider.`
      );
    }
  }

  if (name === 'key-vault') {
    const vault = get(provider, 'vault');

    if (!vault) {
      throw new Error(
        `Invalid provider!! 'provider.vault' must be passed for 'key-vault' provider.`
      );
    }
  }
};

const validators = [isAllowedProvider, validateProvider];

const getValidatedSettings = ({ config }) => {
  validators.forEach(validator => validator(config));

  return Promise.resolve(config);
};

module.exports = { getValidatedSettings };
