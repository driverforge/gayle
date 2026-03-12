const { __mockSend: mockSend } = require('@aws-sdk/client-sts');

const { getAccountId } = require('./get-account-id');

describe('getAccountId', () => {
  it('should get the account Id', () => {
    mockSend.mockImplementation(() =>
      Promise.resolve({
        Account: '12344556',
        Arn: 'eyAreEn',
        UserId: 'useruserId',
      }),
    );

    return expect(getAccountId()).resolves.toEqual('12344556');
  });

  it('should throw if the account Id is missing', () => {
    mockSend.mockImplementation(() =>
      Promise.resolve({
        Arn: 'eyAreEn',
        UserId: 'useruserId',
      }),
    );

    return expect(getAccountId()).rejects.toThrow('Missing accountId');
  });
});
