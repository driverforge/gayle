const { SSMClient } = require('@aws-sdk/client-ssm');
const { getRegion } = require('../../../settings/get-region');

const ssm = new SSMClient({ region: getRegion() });

module.exports = {
  getSsmClient: () => ssm
};
