jest.mock('./get-batch-parameters', () => ({
  getBatchParameters: () =>
    Promise.resolve(['OLD_WORLD', 'OLD_BAR', 'UNCHANGED'])
}));

const {
  __mockSend: mockSend,
  PutParameterCommand
} = require('@aws-sdk/client-ssm');

mockSend.mockImplementation(() => Promise.resolve({ Version: 0 }));

const mockOnComplete = jest.fn();

const { makeSsmStore } = require('./make-ssm-store');

const ssmStore = makeSsmStore();

describe('updateSecrets', () => {
  describe('when an onCompleteHook is provided', () => {
    beforeAll(() =>
      ssmStore.updateSecrets({
        parameters: {
          HELLO: 'WORLD',
          FOO: 'BAR',
          PARAM: 'UNCHANGED'
        },
        onComplete: mockOnComplete
      })
    );

    it('should update secrets in ssm using the default encryption key', () => {
      expect(PutParameterCommand).toBeCalledWith({
        KeyId: 'alias/aws/ssm',
        Name: 'HELLO',
        Overwrite: true,
        Type: 'SecureString',
        Value: 'WORLD'
      });

      expect(PutParameterCommand).toBeCalledWith({
        KeyId: 'alias/aws/ssm',
        Name: 'FOO',
        Overwrite: true,
        Type: 'SecureString',
        Value: 'BAR'
      });
    });

    it('should not update secrets which have not changed', () => {
      expect(PutParameterCommand).not.toBeCalledWith({
        KeyId: 'alias/aws/ssm',
        Name: 'PARAM',
        Overwrite: true,
        Type: 'SecureString',
        Value: 'UNCHANGED'
      });

      expect(
        mockSend.mock.calls.filter(
          call => call[0].commandName === 'PutParameterCommand'
        ).length
      ).toEqual(2);
    });

    it('should run onComplete hook for each parameter', () => {
      expect(mockOnComplete).toBeCalledWith({
        name: 'HELLO',
        value: 'WORLD',
        version: 0
      });

      expect(mockOnComplete).toBeCalledWith({
        name: 'FOO',
        value: 'BAR',
        version: 0
      });
    });
  });

  describe('when an onComplete hook is not provided', () => {
    it('should still persist the secret in ssm', () => {
      PutParameterCommand.mockClear();
      mockSend.mockClear();
      mockSend.mockImplementation(() => Promise.resolve({ Version: 0 }));

      expect.assertions(1);

      return ssmStore
        .updateSecrets({
          parameters: {
            HELLO: 'WORLD'
          }
        })
        .then(() => {
          expect(PutParameterCommand).toBeCalledWith({
            KeyId: 'alias/aws/ssm',
            Name: 'HELLO',
            Overwrite: true,
            Type: 'SecureString',
            Value: 'WORLD'
          });
        });
    });
  });
});
