# Technitium DNS Server API Reference

This document describes the Technitium DNS Server API endpoints used by the webhook.

## Base URL

The base URL for all API requests is configured via the `TECHNITIUM_URL` environment variable.

Default: `http://localhost:5380`

## Authentication

All API requests require authentication using either:

1. **API Token**: Passed as `token` parameter
2. **Session Token**: Obtained via login and passed as `token` parameter

### Login

**Endpoint**: `POST /api/user/login`

**Parameters**:
- `user` (string): Username
- `pass` (string): Password

**Response**:
```json
{
  "status": "ok",
  "token": "abc123def456..."
}
```

**Example**:
```bash
curl -X POST http://localhost:5380/api/user/login \
  -d "user=admin" \
  -d "pass=admin"
```

## Zones

### List Zones

**Endpoint**: `GET /api/zones/list`

**Parameters**:
- `token` (string): Authentication token

**Response**:
```json
{
  "status": "ok",
  "zones": [
    {
      "name": "example.com",
      "type": "Primary",
      "internal": false,
      "disabled": false
    }
  ]
}
```

**Example**:
```bash
curl "http://localhost:5380/api/zones/list?token=TOKEN"
```

### Create Zone

**Endpoint**: `GET /api/zones/create`

**Parameters**:
- `token` (string): Authentication token
- `zone` (string): Zone name (e.g., "example.com")
- `type` (string): Zone type ("Primary", "Secondary", "Stub", "Forwarder")

**Response**:
```json
{
  "status": "ok"
}
```

**Example**:
```bash
curl "http://localhost:5380/api/zones/create?token=TOKEN&zone=example.com&type=Primary"
```

### Delete Zone

**Endpoint**: `GET /api/zones/delete`

**Parameters**:
- `token` (string): Authentication token
- `zone` (string): Zone name

**Response**:
```json
{
  "status": "ok"
}
```

## Records

### Get Records

**Endpoint**: `GET /api/zones/records/get`

**Parameters**:
- `token` (string): Authentication token
- `domain` (string): Domain name
- `zone` (string, optional): Zone name
- `listZone` (boolean, optional): List all records in zone

**Response**:
```json
{
  "status": "ok",
  "records": [
    {
      "name": "www.example.com",
      "type": "A",
      "ttl": 3600,
      "rData": {
        "ipAddress": "192.0.2.1"
      },
      "disabled": false
    }
  ]
}
```

**Example**:
```bash
curl "http://localhost:5380/api/zones/records/get?token=TOKEN&domain=example.com&listZone=true"
```

### Add Record

**Endpoint**: `GET /api/zones/records/add`

**Parameters**:
- `token` (string): Authentication token
- `domain` (string): Record name
- `zone` (string): Zone name
- `type` (string): Record type (A, AAAA, CNAME, TXT, etc.)
- `ttl` (integer): Time to live in seconds
- Record-specific parameters (see below)

**Record Types**:

#### A Record
- `ipAddress` (string): IPv4 address

**Example**:
```bash
curl "http://localhost:5380/api/zones/records/add?token=TOKEN&domain=www.example.com&zone=example.com&type=A&ttl=3600&ipAddress=192.0.2.1"
```

#### AAAA Record
- `ipAddress` (string): IPv6 address

**Example**:
```bash
curl "http://localhost:5380/api/zones/records/add?token=TOKEN&domain=www.example.com&zone=example.com&type=AAAA&ttl=3600&ipAddress=2001:db8::1"
```

#### CNAME Record
- `cname` (string): Canonical name

**Example**:
```bash
curl "http://localhost:5380/api/zones/records/add?token=TOKEN&domain=alias.example.com&zone=example.com&type=CNAME&ttl=3600&cname=www.example.com"
```

#### TXT Record
- `text` (string): Text content

**Example**:
```bash
curl "http://localhost:5380/api/zones/records/add?token=TOKEN&domain=_acme-challenge.example.com&zone=example.com&type=TXT&ttl=300&text=verification-string"
```

#### MX Record
- `preference` (integer): Priority (lower is higher priority)
- `exchange` (string): Mail server hostname

**Example**:
```bash
curl "http://localhost:5380/api/zones/records/add?token=TOKEN&domain=example.com&zone=example.com&type=MX&ttl=3600&preference=10&exchange=mail.example.com"
```

**Response**:
```json
{
  "status": "ok"
}
```

### Update Record

**Endpoint**: `GET /api/zones/records/update`

**Parameters**:
- `token` (string): Authentication token
- `domain` (string): Record name
- `zone` (string): Zone name
- `type` (string): Record type
- `ttl` (integer): New TTL
- Old record parameters (prefixed with `old`)
- New record parameters (prefixed with `new`)

**Example**:
```bash
curl "http://localhost:5380/api/zones/records/update?token=TOKEN&domain=www.example.com&zone=example.com&type=A&ttl=7200&oldIpAddress=192.0.2.1&newIpAddress=192.0.2.2"
```

### Delete Record

**Endpoint**: `GET /api/zones/records/delete`

**Parameters**:
- `token` (string): Authentication token
- `domain` (string): Record name
- `zone` (string): Zone name
- `type` (string): Record type
- Record-specific parameters to identify the exact record

**Example**:
```bash
curl "http://localhost:5380/api/zones/records/delete?token=TOKEN&domain=www.example.com&zone=example.com&type=A&ipAddress=192.0.2.1"
```

**Response**:
```json
{
  "status": "ok"
}
```

## Error Responses

All endpoints return errors in the following format:

```json
{
  "status": "error",
  "errorMessage": "Description of the error"
}
```

**Common Error Cases**:
- Invalid authentication token
- Zone already exists
- Zone not found
- Record not found
- Invalid parameters

## Webhook Implementation

### Client Usage

The webhook implements a Go client that wraps these API calls:

```go
// Initialize client
client := technitium.NewClient(baseURL, token)
// or
client, err := technitium.NewClientWithAuth(baseURL, username, password)

// List zones
zones, err := client.ListZones()

// Create zone
err := client.CreateZone("example.com")

// Get records
records, err := client.GetRecords("example.com")

// Add record
err := client.AddRecord("example.com", "www.example.com", "A", 3600, "192.0.2.1")

// Delete record
err := client.DeleteRecord("example.com", "www.example.com", "A", "192.0.2.1")
```

### Record Type Mapping

| DNS Record Type | external-dns | Technitium | Webhook Support |
|----------------|--------------|------------|-----------------|
| A              | ✓            | ✓          | ✓               |
| AAAA           | ✓            | ✓          | ✓               |
| CNAME          | ✓            | ✓          | ✓               |
| TXT            | ✓            | ✓          | ✓               |
| MX             | ✓            | ✓          | ⚠️ (TODO)       |
| SRV            | ✓            | ✓          | ⚠️ (TODO)       |
| NS             | ✓            | ✓          | ❌              |
| PTR            | ✓            | ✓          | ❌              |

## Rate Limiting

Technitium does not enforce strict rate limits, but consider implementing client-side rate limiting for production deployments with many records.

## Recommended Practices

1. **Use API Tokens**: Prefer API tokens over username/password for better security
2. **Cache Zone List**: Cache the zone list to reduce API calls
3. **Batch Operations**: Group related operations when possible
4. **Error Handling**: Implement retry logic for transient failures
5. **Logging**: Log all API interactions for debugging

## Additional Resources

- [Official Technitium API Documentation](https://github.com/TechnitiumSoftware/DnsServer/blob/master/APIDOCS.md)
- [Technitium DNS Server](https://technitium.com/dns/)
- [GitHub Repository](https://github.com/TechnitiumSoftware/DnsServer)
