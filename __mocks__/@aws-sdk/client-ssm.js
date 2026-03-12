const mockSend = jest.fn();

const SSMClient = jest.fn(() => ({
  send: mockSend,
}));

const GetParametersCommand = jest.fn((input) => ({
  commandName: 'GetParametersCommand',
  input,
}));

const GetParametersByPathCommand = jest.fn((input) => ({
  commandName: 'GetParametersByPathCommand',
  input,
}));

const PutParameterCommand = jest.fn((input) => ({
  commandName: 'PutParameterCommand',
  input,
}));

const DeleteParametersCommand = jest.fn((input) => ({
  commandName: 'DeleteParametersCommand',
  input,
}));

module.exports = {
  SSMClient,
  GetParametersCommand,
  GetParametersByPathCommand,
  PutParameterCommand,
  DeleteParametersCommand,
  __mockSend: mockSend,
};
