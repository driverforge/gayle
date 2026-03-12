const mockSend = jest.fn(() => Promise.resolve());

const CloudFormationClient = jest.fn(() => ({
  send: mockSend,
}));

const DescribeStacksCommand = jest.fn((input) => ({
  _commandName: 'DescribeStacksCommand',
  input,
}));

module.exports = {
  CloudFormationClient,
  DescribeStacksCommand,
  __mockSend: mockSend,
};
