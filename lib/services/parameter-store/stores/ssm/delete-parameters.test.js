const { __mockSend: mockSend } = require('@aws-sdk/client-ssm');

mockSend.mockImplementation(() => Promise.resolve({}));

const { makeSsmStore } = require('./make-ssm-store');

describe('deleteParameters', () => {
  it('should delete parameters in batches', async () => {
    const parameterNames = [...Array(35).keys()].map(key => `/test/${key}`);

    const ssm = makeSsmStore();

    await ssm.deleteParameters({ parameterNames });

    expect(mockSend.mock.calls.length).toEqual(4);
  });
});
