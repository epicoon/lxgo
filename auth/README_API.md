# API
* [/auth `POST`](#r1)
* [/user-data `GET`](#r2)
* [/user-data `POST`](#r7)
* [/tokens `POST`](#r3)
* [/refresh `POST`](#r4)
* [/logout `POST`](#r5)
* [/clients `POST`](#r8)
* [/admin/clients `DELETE`](#r9)
* [Error responses](#r6)

---
## <a name="r1">/auth</a>

### Path: `/auth`
### Description: redirect to the authorization service form
### Method: `POST`

### Request Params
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| response_type | string | yes | authentication type, available values: 'code' |
| client_id | uint | yes | client application identifier |
| redirect_uri | string | yes | where to redirect the user after authorization; must exactly match the URI registered for this client, otherwise the request is rejected with `400 Bad Request` |
| state | string | yes | unique string for protection against CSRF. Need to be generated and saved (possibly in the session) to check when returning from the authorization form in the client application |
| scope | string | no | requested access level: `profile` or `profile:data` (see [/user-data](#r2) below for what each one grants); defaults to `profile` (the narrowest one) if omitted; an unknown value is rejected with `400 Bad Request` |

### Make request by redirect. The way to do this with `POST` method in browser:
```js
const form = document.createElement("form");
form.method = "POST";
form.action = `${auth_server_url}/auth`;

const params = {
    response_type: 'code',
    client_id,
    redirect_uri,
    state
};

for (const key in params) {
    if (params.hasOwnProperty(key)) {
        const input = document.createElement("input");
        input.type = "hidden";
        input.name = key;
        input.value = params[key];
        form.appendChild(input);
    }
}

document.body.appendChild(form);
form.submit();
```

---
## <a name="r2">/user-data</a>

### Path: `/user-data`
### Description: getting client data by token
### Method: `GET`

### Request Headers
* Authorization: Bearer <access_token>

### Request Params
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| client_id | uint | yes | client application identifier |

### Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals true |
| login | string | yes | unique user login in the system |
| data | string | yes | JSON-encoded user data previously reported by the client application via [POST /user-data](#r7); empty if none was reported yet, or if the access token's scope is not `profile:data` (a `profile`-scoped token gets `login` only) |

### Failed Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals false |
| error_code | int | yes | number code of the issue occured |
| error_message | string | yes | explanation of the issue occured |

---
## <a name="r7">/user-data (write)</a>

### Path: `/user-data`
### Description: store arbitrary data about the current user on behalf of the client application, retrievable later via [GET /user-data](#r2)
### Method: `POST`

### Request Headers
* Authorization: Bearer <access_token>

### Request Params
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| client_id | uint | yes | client application identifier |
| data | string | yes | arbitrary JSON-encoded data to store for the current user; replaces whatever was stored before for this (user, client) pair |

Requires an access token with `profile:data` scope (same as [GET /user-data](#r2)) - a `profile`-scoped token is
rejected with `403 Forbidden`, error code 1018.

### Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals true |

### Failed Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals false |
| error_code | int | yes | number code of the issue occured |
| error_message | string | yes | explanation of the issue occured |

---
## <a name="r3">/tokens</a>

### Path: `/tokens`
### Description: exchange code for a pair of tokens (access_token, refresh_token)
### Method: `POST`

### Request Params
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| code | string | yes | unique string for exchange for tokens (redirect endpoint link) |
| client_id | uint | yes | client application identifier |
| client_secret | string | yes | unique string for client application verification |

### Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals true |
| access_token | string | yes | access token |
| refresh_token | string | yes | token for refresh of a pair of tokens |
| access_token_expired | int64 | yes | UNIX timestamp in seconds when the token will expire |
| refresh_token_expired | int64 | yes | UNIX timestamp in seconds when the token will expire |
| scope | string | yes | access level actually granted for this pair of tokens (the one that was requested at [/auth](#r1)) |

### Failed Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals false |
| error_code | int | yes | number code of the issue occured |
| error_message | string | yes | explanation of the issue occured |

---
## <a name="r4">/refresh</a>

### Path: `/refresh`
### Description: reissue of a pair of tokens
### Method: `POST`

### Request Params
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| grant_type | string | yes | method for reissuing tokens, currently implemented 'refresh_token' |
| client_id | uint | yes | client application identifier |
| client_secret | string | yes | unique string for client application verification |
| refresh_token | string | yes | token for refresh of a pair of tokens |
| scope | string | no | optionally narrow the scope of the reissued tokens; must not exceed the scope already granted (broadening is rejected with `400 Bad Request`, error code 1017); omit to keep the current scope unchanged |

### Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals true |
| access_token | string | yes | access token |
| refresh_token | string | yes | token for refresh of a pair of tokens |
| access_token_expired | int64 | yes | UNIX timestamp in seconds when the token will expire |
| refresh_token_expired | int64 | yes | UNIX timestamp in seconds when the token will expire |
| scope | string | yes | access level actually granted for this (reissued) pair of tokens |

### Failed Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals false |
| error_code | int | yes | number code of the issue occured |
| error_message | string | yes | explanation of the issue occured |

---
## <a name="r5">/logout</a>

### Path: `/logout`
### Description: to log the user out of the system
### Method: `POST`

### Request Headers
* Authorization: Bearer <access_token>

### Request Params
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| client_id | uint | yes | client application identifier |

### Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals true |

### Failed Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals false |
| error_code | int | yes | number code of the issue occured |
| error_message | string | yes | explanation of the issue occured |

---
## <a name="r8">/clients (register)</a>

### Path: `/clients`
### Description: self-service registration of a new OAuth client - open, no authentication required (same as e.g. Google/GitHub let any developer register their own app)
### Method: `POST`

### Request Params
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| redirect_uri | string | yes | where to redirect the user after authorization; the exact value this client will send to [/auth](#r1) |

### Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals true |
| id | uint | yes | new client's identifier |
| secret | string | yes | new client's secret - shown once here, store it now |

### Failed Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals false |
| error_code | int | yes | number code of the issue occured |
| error_message | string | yes | explanation of the issue occured |

---
## <a name="r9">/admin/clients (delete)</a>

### Path: `/admin/clients`
### Description: delete an OAuth client - the first (and so far only) operation of the [admin API](https://github.com/epicoon/lxgo/tree/master/auth/README.md#admin-api)
### Method: `DELETE`

### Request Headers
* Authorization: Bearer <access_token>

The access token must have been issued through the client configured as `Settings.AdminClientID` (see the main
README's ["Deploying the service"](https://github.com/epicoon/lxgo/tree/master/auth/README.md#deploy)) for a `User`
that has an admin record - a token from any other client is rejected as if it didn't exist at all (`401`, error code
1004), even for the same underlying `User`, and a non-admin `User`'s token (even through the right client) is
rejected with `403`, error code 1019.

### Request Params
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| id | uint | yes | identifier of the client to delete |

### Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals true |

### Failed Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals false |
| error_code | int | yes | number code of the issue occured |
| error_message | string | yes | explanation of the issue occured |

---
## <a name="r6">Error responses</a>

### Response body structure in case of error:
```yaml
success: false
error_code: int
error_message: string
```

### Error Code Table
| Response code | Error code | Error details                               |
| ------------- | ---------- | ------------------------------------------- |
| 400           | 1001       | Verification code is not valid              |
| 400           | 1002       | Invalid or missing refresh token            |
| 400           | 1003       | Invalid or missing client id                |
| 401           | 1004       | Token not found                             |
| 401           | 1005       | Token expired                               |
| 401           | 1006       | Invalid or expired token                    |
| 401           | 1007       | Request issue: authorization header missing |
| 401           | 1007       | Request issue: invalid authorization scheme |
| 404           | 1008       | Client not found                            |
| 404           | 1009       | User not found                              |
| 500           | 1010       | Something went wrong                        |
| 500           | 1011       | Error occured                               |
| 400           | 1017       | Requested scope exceeds the scope already granted |
| 403           | 1018       | Token scope does not allow storing user data |
| 403           | 1019       | Not an admin                                |
