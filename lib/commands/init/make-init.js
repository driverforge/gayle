const get = require('lodash/get');

// TODO: create helper method on settings service to validate provider
const makeInit = ({ settingsService }) => {
  const initSsm = () => Promise.resolve();

  return async () => {
    const settings = await settingsService.getSettings();

    const providerName = get(settings, 'provider.name');

    if (providerName === 'key-vault') {
      return Promise.resolve();
    }

    return initSsm();
  };
};

module.exports = {
  makeInit,
};
