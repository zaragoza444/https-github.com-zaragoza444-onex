# OneX Marketing Website

Static marketing site for **onexproduction.com**.

## Preview locally

```powershell
cd website
python -m http.server 8080
```

Open http://127.0.0.1:8080

## Production

Served at `/` by nginx when using `docker compose -f docker-compose.prod.yml --profile proxy up -d`.  
Wallet remains at `/wallet/`.

## Business email

See `docs/BUSINESS-EMAIL.md` and `configs/business-email.json`.

| Address | Purpose |
|---------|---------|
| hello@onexproduction.com | General |
| business@onexproduction.com | Partnerships |
| support@onexproduction.com | Support |
| security@onexproduction.com | Security |
