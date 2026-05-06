# How one can be sure that lovely 🌸🌸🌸 is not producing 💩💩💩

We utilize different testing layers:

- **API (Integration Tests)**: These validate the request-response cycle and any resulting side effects. You send a payload, check the response, and verify the system behaves as expected. (See existing tests for examples).
- **Services**: These cover internal business logic that isn't always exposed via the API. For example, if you need to offboard users or clean up accounts based on specific criteria, test the logic here.
- **Workers**: Our background processes that run on a schedule. Here you can seed your data and manually trigger a job to make sure it actually does its thing.

## Setup
Each of these layers requires a real database connection. Please refer to `../SETUP.md` for help with the test database and its specific configuration.