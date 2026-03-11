const get = require('lodash/get');
const {
  CloudFormationClient,
  DescribeStacksCommand
} = require('@aws-sdk/client-cloudformation');
const chalk = require('chalk');
const { log, logWarning } = require('../../utils/logger');

const { getRegion } = require('../settings/get-region');

const cloudformation = new CloudFormationClient({ region: getRegion() });

const readCfOutputs = async ({ stackName }) => {
  log(chalk.cyan(`Getting stack outputs for: [${stackName}]`));

  if (!stackName) {
    throw new Error('Please specify stackName for "stacks"');
  }

  const params = { StackName: stackName };

  return cloudformation
    .send(new DescribeStacksCommand(params))
    .then(res => get(res, 'Stacks.0.Outputs') || [])
    .then(res =>
      res.reduce((acc, output) => {
        const key = get(output, 'OutputKey');
        const value = get(output, 'OutputValue');

        return { ...acc, [key]: value };
      }, {})
    )
    .catch(() => {
      logWarning(`Could not find stack outputs for: [${stackName}]`);
      return {};
    });
};

const getOutputs = ({ stackNames = [] }) =>
  Promise.all(
    stackNames.map(stackName => readCfOutputs({ stackName }))
  ).then(outputs =>
    outputs.reduce((acc, output) => ({ ...acc, ...output }), {})
  );

module.exports = {
  getOutputs
};
