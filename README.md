# ec2-stopwatch

USE THIS: https://docs.aws.amazon.com/solutions/latest/ec2-scheduler/welcome.html

NOTE: This is deprecated and will no longer be supported. It's many years old at this point and honestly, putting something that serves this purpose in lambda would be much better.
The concept is fine, but this implementation should probably be abandoned for something designed to run on lambda.

HTTP API for managing ec2 instances. Scheduling periods when an instance should be on and off, and also setting instance expiration dates (not yet implemented)

## Dependencies
* [Redis](http://redis.io/)
* [Hashicorp Vault](https://vaultproject.io/)

## Environment Variables
Variable Name | Description
------------ | -------------
VAULT_ADDR | The address where stopwatch will find your vault server (e.g. http://127.0.0.1:8200)
VAULT_TOKEN | The Vault Root Token
VAULT_KEY_1 | Vault Key 1 to Unseal
VAULT_KEY_2 | Vault Key 2 to Unseal
VAULT_KEY_3 | Vault Key 3 to Unseal
REDIS_HOST_PORT | The host:port for redis (e.g. localhost:6379)
REDIS_PASSWORD | The password used to AUTH to redis
SENDGRID_API_TOKEN | Sendgrid API Token for Email
EMAIL_FROM_ADDRESS | When the application sends email, the From address
STOPWATCH_URL | When the application sends links (for email verification) the url at which the user can find stopwatch
AWS_ACCESS_KEY | The Access Key used to make AWS API Calls
AWS_SECRET_KEY | The Secret Key used to make AWS API Calls

## Endpoints

### `POST /register` User registration

Request Body:
```json
{
  "email": "some@email.com"
}
```

### `GET /verify/:token` Email verification

## Private Endpoints
Private endpoints require the `Stopwatch-Api-Token` header to be included with all requests. The value of this header should be your stopwatch api token.

### `GET /private/user` Get user email from token

### `POST /private/aws/secrets` Store AWS Access/Secret Keys
Request Body:
```json
{
  "access_key_id": "fill_in",
  "secret_key_id": "secrets"
}
```

### `POST /private/aws/schedule` Create schedule for AWS EC2 instance
```json
{
  "access_key_id": "stuff",
  "instance_id": "placeholder",
  "start": {
    "minute": "38",
    "hour": "15",
    "day_of_month": "*",
    "month": "*",
    "day_of_week": "1-5"
  },
  "end": {
    "minute": "0",
    "hour": "20",
    "day_of_month": "*",
    "month": "*",
    "day_of_week": "1-5"
  },
  "region": "any valid region",
  "ddns": {
    "enabled": "true",
    "domain": "your.domain.com",
    "hosted_zone_id": "placeholder"
  }
}
```

## TODOs
* Add support for multiple data stores (not just redis)
  * Write data abstraction layer and backend storage drivers for each storage backend
* Add support for more cloud providers (joyent, azure, google cloud) not just AWS
