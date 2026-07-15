const { get, property, isEmpty } = require('lodash');
const chalk = require('chalk');
const { log } = require('../../utils/logger');

const maskValue = (value) => value.replace(/\S(?=\S{4})/g, '*');

const makeCleanup =
  ({ parameterStore, settingsService }) =>
  async ({ dryRun } = { dryRun: false }) => {
    const settings = await settingsService.getSettings();
    const configPath = get(settings, 'config.path');
    const secretPath = get(settings, 'secret.path');

    const { configParameters = [], secretParameters = [] } = settings;
    const declaredParameters = [...configParameters, ...secretParameters];

    // An empty or misparsed configuration would classify every remote
    // parameter under the configured paths as unused and delete the lot.
    if (isEmpty(declaredParameters)) {
      throw new Error(
        'Cleanup refused: the configuration declares no config or secret keys. ' +
          'Pruning against an empty declaration would delete every remote parameter under the configured paths.',
      );
    }

    const parameters = await Promise.all([
      parameterStore.getAllParameters({ path: configPath }),
      parameterStore.getAllParameters({ path: secretPath }),
    ]).then(([configs, secrets]) => [...configs, ...secrets]);

    const unusedParameters = parameters.filter(
      ({ Name }) => !declaredParameters.includes(Name),
    );

    if (isEmpty(unusedParameters)) {
      log(chalk.gray('Cleanup --> No unused parameters'));
      return Promise.resolve();
    }

    if (dryRun) {
      log(chalk.gray('Cleanup --> Parameters to be deleted: '));
      return unusedParameters.map(({ Name, Value, Type }) => {
        const shouldMask = Type === 'SecureString';
        return log(
          chalk.gray(
            `Cleanup --> Name: ${Name} | Value: [${
              shouldMask ? maskValue(Value) : Value
            }]`,
          ),
        );
      });
    }

    unusedParameters.forEach(({ Name }) =>
      log(chalk.yellow(`Cleanup --> Deleting unused parameter: ${Name}`)),
    );

    return parameterStore.deleteParameters({
      parameterNames: unusedParameters.map(property('Name')),
    });
  };

module.exports = { makeCleanup };
