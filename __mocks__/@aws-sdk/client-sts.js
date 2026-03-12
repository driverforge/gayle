const mockSend = jest.fn(() =>
  Promise.resolve({
    Account: '1234',
    Arn: 'mock-arn',
    UserId: 'mock-user-id',
  }),
);

const STSClient = jest.fn(() => ({
  send: mockSend,
}));

const GetCallerIdentityCommand = jest.fn((input) => ({
  _commandName: 'GetCallerIdentityCommand',
  input,
}));

module.exports = {
  STSClient,
  GetCallerIdentityCommand,
  __mockSend: mockSend,
};
