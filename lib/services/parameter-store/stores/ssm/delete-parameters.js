const chunk = require('lodash/chunk');
const { DeleteParametersCommand } = require('@aws-sdk/client-ssm');
const { getSsmClient } = require('./get-ssm-client');

const deleteParameters = ({ parameterNames }) => {
  const ssm = getSsmClient();

  const chunks = chunk(parameterNames, 10);
  const promises = chunks.map((chunkedParameterNames) =>
    ssm.send(
      new DeleteParametersCommand({
        Names: chunkedParameterNames,
      }),
    ),
  );
  return Promise.all(promises);
};

module.exports = { deleteParameters };
