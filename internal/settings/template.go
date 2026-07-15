package settings

// GenerateTemplate is the example gayle.yml written by `gayle generate` —
// byte-identical to the Node CLI's js-yaml dump of its defaultConfig.
const GenerateTemplate = `service: my-service
provider:
  name: ssm
config:
  path: /${stage}/config
  defaults:
    DB_NAME: my-database
    DB_HOST: 3200
  required:
    DB_TABLE: some database table name for ${stage}
secret:
  path: /${stage}/secret
  required:
    DB_PASSWORD: secret database password
`
