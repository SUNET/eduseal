# JWT

## URL

prod: auth.sunet.se

test: auth-test.sunet.se

## Function certificates

the service is trusting these certificates,

* SITHS e-id funktionscertifikat (Inera)
* E-identitet för offentlig sektor – EFOS (Försäkringskassan)
* ExpiTrust EID CA V4[9] (tidigare Steria AB e-Tjänstelegitimationer Kort CA v2 hos Expisoft AB

## Making your own cert/key

## Request JWT token

POST: /transaction

### Fingerprint

```
{
 "access_token": [
  {
   "flags": [
    "bearer"
   ]
  }
 ],
 "client": {
  "key": {
   "proof": "mtls",
   "cert#S256": "<fingerprint>"
  }
 }
}
```

### client key

```
{
 "access_token": [
  {
   "flags": [
    "bearer"
   ],
   "access": [
    {
     "type": "eduseal"
    }
   ]
  }
 ],
 "client": {
  "key": "<key_name_in_service_config>"
 }
}
```

## Claims

```
{
    "auth_source":"config",
    "exp":1720791975,
    "iat":1720788375,
    "iss":"https://auth-test.sunet.se",
    "nbf":1720788375,
    "organization_id":"<claim_in_server_config>",
    "requested_access":[
        {"type":"eduseal"}
        ],
    "source":"config",
    "sub":"<key_name_in_service_config>",
    "version":1}
```
