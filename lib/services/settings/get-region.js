const DEFAULT_REGION = 'us-east-1';

const getRegion = () =>
  process.env.AWS_REGION || process.env.AWS_DEFAULT_REGION || DEFAULT_REGION;

module.exports = {
  getRegion
};
