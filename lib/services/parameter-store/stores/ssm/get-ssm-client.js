const { SSMClient } = require('@aws-sdk/client-ssm');

const ssm = new SSMClient({ region: 'us-east-1' });

module.exports = {
  getSsmClient: () => ssm
};
