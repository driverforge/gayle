const SEPARATOR = '--';

const toKeyVaultName = (internalName) => {
  const parts = internalName.split('/');
  const service = parts.slice(0, -1).join('/');
  const key = parts[parts.length - 1];
  const kvKey = key.replace(/_/g, '-');
  return `${service}${SEPARATOR}${kvKey}`;
};

const fromKeyVaultName = (kvName) => {
  const separatorIndex = kvName.indexOf(SEPARATOR);
  if (separatorIndex === -1) {
    return kvName;
  }
  const service = kvName.substring(0, separatorIndex);
  const kvKey = kvName.substring(separatorIndex + SEPARATOR.length);
  const key = kvKey.replace(/-/g, '_');
  return `${service}/${key}`;
};

const extractKeyName = (internalName) => {
  const parts = internalName.split('/');
  return parts[parts.length - 1];
};

module.exports = {
  toKeyVaultName,
  fromKeyVaultName,
  extractKeyName,
  SEPARATOR,
};
