const {
  toKeyVaultName,
  fromKeyVaultName,
  extractKeyName,
} = require('./key-utils');

describe('key-utils', () => {
  describe('toKeyVaultName', () => {
    it('should convert internal path format to key vault name', () => {
      expect(toKeyVaultName('graph/DATABASE_URL')).toEqual(
        'graph--DATABASE-URL',
      );
    });

    it('should handle keys without underscores', () => {
      expect(toKeyVaultName('graph/HOSTNAME')).toEqual('graph--HOSTNAME');
    });

    it('should handle multiple underscores', () => {
      expect(toKeyVaultName('graph/MY_DB_HOST')).toEqual('graph--MY-DB-HOST');
    });
  });

  describe('fromKeyVaultName', () => {
    it('should convert key vault name to internal path format', () => {
      expect(fromKeyVaultName('graph--DATABASE-URL')).toEqual(
        'graph/DATABASE_URL',
      );
    });

    it('should handle keys without hyphens after separator', () => {
      expect(fromKeyVaultName('graph--HOSTNAME')).toEqual('graph/HOSTNAME');
    });

    it('should handle names without separator', () => {
      expect(fromKeyVaultName('noseparator')).toEqual('noseparator');
    });
  });

  describe('extractKeyName', () => {
    it('should extract the key name from an internal path', () => {
      expect(extractKeyName('graph/DATABASE_URL')).toEqual('DATABASE_URL');
    });
  });
});
