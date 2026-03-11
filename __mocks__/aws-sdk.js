const S3 = jest.fn();
const SecretsManager = jest.fn();
const CloudFormation = jest.fn(() => ({
  describeStacks: jest.fn(() => ({
    promise: () => Promise.resolve()
  }))
}));
const KMS = jest.fn(() => {});
const SSM = jest.fn(() => {});
const SNS = jest.fn(() => {});
const STS = jest.fn(() => ({
  getCallerIdentity: () => ({
    promise: () =>
      Promise.resolve({
        Account: '1234'
      })
  })
}));

const config = {
  setPromisesDependency: jest.fn(),
  update: jest.fn()
};

module.exports = {
  config,
  CloudFormation,
  SecretsManager,
  STS,
  KMS,
  SNS,
  SSM,
  S3
};
