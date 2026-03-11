const get = require('lodash/get');
const { STSClient, GetCallerIdentityCommand } = require('@aws-sdk/client-sts');
const { getRegion } = require('./get-region');

const sts = new STSClient({ region: getRegion() });

const getAccountId = () =>
  sts
    .send(new GetCallerIdentityCommand({}))
    .then(res => {
      const accountId = get(res, 'Account');
      if (!accountId) {
        throw new Error('Missing accountId');
      }

      return accountId;
    })
    .catch(error => {
      throw new Error(error.message);
    });

module.exports = {
  getAccountId
};
