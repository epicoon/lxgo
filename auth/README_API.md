# API
* [/auth `POST`](#r1)
* [/user-data `GET`](#r2)
* [/tokens `POST`](#r3)
* [/refresh `POST`](#r4)
* [/logout `POST`](#r5)
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
| redirect_uri | string | yes | where to redirect the user after authorization |
| state | string | yes | unique string for protection against CSRF. Need to be generated and saved (possibly in the session) to check when returning from the authorization form in the client application |

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
| data | string | yes | user data previously reported by the client application (! not implemented !) |

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

### Response Body
| field name    | type   | obligate | description |
| ------------- | ------ | -------- | ----------- |
| success | bool | yes | equals true |
| access_token | string | yes | access token |
| refresh_token | string | yes | token for refresh of a pair of tokens |
| access_token_expired | int64 | yes | UNIX timestamp in seconds when the token will expire |
| refresh_token_expired | int64 | yes | UNIX timestamp in seconds when the token will expire |

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
## <a name="r6">Error responses</a>

### Response body structure in case of error:
```yaml
success: false
error_code: int value from range 1001 - 1011
error_message: string
```

### Error Code Table
| Response code | Error code | Error datails                               |
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
