# dsn

Written to anticipate optimistically forwarded requests for an on prem Sentry (8.13)
store endpoint. This is used to send JSON event payloads to Sentry. It is located at: /api/{projectID}/store/.

It will not handle forwarded requests to the sentry API: /api/0/ 



    