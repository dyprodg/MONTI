# Monitoring Access

## Grafana

**URL:** http://3.69.80.81:3000

**Credentials:**
- Username: `admin`


**Dashboard:** MONTI Overview (auto-provisioned)

## Prometheus

**URL:** http://3.69.80.81:9090

No authentication required.

## Retrieving Credentials

If you need to fetch the password again:

```bash
aws ssm get-parameter --name "/monti/grafana/admin-password" --with-decryption --query 'Parameter.Value' --output text --region eu-central-1
```

## Ports Reference

| Service    | Port | Description          |
|------------|------|----------------------|
| Grafana    | 3000 | Dashboards & alerts  |
| Prometheus | 9090 | Metrics & queries    |
| cAdvisor   | 8082 | Container metrics    |
