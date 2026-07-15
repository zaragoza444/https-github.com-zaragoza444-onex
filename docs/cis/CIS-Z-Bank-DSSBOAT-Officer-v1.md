# CIS — Z Bank · DSSBOaT Corporate Officer v1.0

**Component Integration Specification — Client due-diligence profile**

| Field | Value |
|-------|-------|
| Document ID | `CIS-Z-BANK-DSSBOAT-OFFICER-v1` |
| Version | 1.0 |
| Status | Draft |
| Platform | OneX Bridge / Z Bank Online |
| Source CIS PDF | `docs/cis/DSSBOAT_CIS_2026.pdf` |
| Parent CIS | `CIS-Z-Bank-Online-v1.md` |
| Date of source CIS | 21 February 2025 |

---

## 1. Purpose and scope

This CIS onboard **DSSBOaT Ltd Trading as DS Group** as a Z Bank corporate client and defines the **authorized officer** (signatory) who may operate Z Bank accounts with **PIN + wet/typed signature** authentication.

Prepared with reference to Articles 2–5 of the Due Diligence Convention, the Criminal Justice (Money Laundering and Terrorist Financing) Act 2010 (as amended), and Central Bank of Ireland AML/CFT guidance, for identity and funds-origin verification with banks and other financial institutions — subject to agreement by the Client and all individuals named herein, and indefinite confidentiality obligations.

**In scope**

- Corporate identity of DSSBOaT Ltd (Ireland)
- Officer / signatory profile (Bernard Greeff Niehaus, CEO)
- Z Bank officer APIs: register/list/status, PIN + signature verify, authorized transfers
- Linkage to Z Bank M1–M4 ledger accounts

**Out of scope**

- Storing passport scan images in ledger JSON (PDF remains the evidentiary attachment)
- Full third-party KYC vendor orchestration
- Nova Bank branding or seeds

---

## 2. Client corporate information

| Field | Value |
|-------|-------|
| Company name | **DSSBOaT Ltd** Trading as **DS Group** |
| Country of incorporation | Ireland |
| Corporate registration | **725265** |
| Date of registration | 2 September 2022 |
| Company address | 92 George’s Street Lower, Dún Laoghaire, Dublin, Ireland, A96 VR66 |

---

## 3. Officer and signatory

| Field | Value |
|-------|-------|
| Officer name | **Bernard Greeff Niehaus** |
| Officer position | CEO / Founder |
| Passport nationality | New Zealander |
| Passport number | LK986067 |
| Passport date of issue | 28 April 2017 |
| Passport date of expiry | 28 April 2027 |
| Date of birth | 29 May 1982 |
| Residential address | 340 Whitford Road, Howick, Auckland NZ |
| Email | bniehaus@dssalus.com |
| Phone | +447425940224 |
| System officer ID | `dssboat-officer-bneihaus` |

Affirmation (per source CIS): the officer affirms under penalty of perjury that the information and attachments are true in all material respects as of the date of signing.

---

## 4. Officer bank authentication (PIN & signature)

Z Bank officer operations require **both**:

| Factor | Rule | Storage |
|--------|------|---------|
| **PIN** | 4–8 digits | SHA-256 hash (`onex-zbank-officer-pin:` + pin + `:` + officer salt) — never stored in plaintext |
| **Signature** | Typed signature passphrase / signature code (≥ 8 chars) | SHA-256 hash (`onex-zbank-officer-sig:` + normalized signature + `:` + officer salt) |

Authorized transfers (`POST /bridge/bank/officer/transfer`) must include `officerId`, `pin`, and `signature`. Verification fails closed if either factor is missing or mismatched.

Seed config: `configs/zbank-officers.dssboat.example.json`  
Env override for demo PIN/signature at first seed (rotate in production):

```env
ONEX_ZBANK_OFFICERS_FILE=configs/zbank-officers.dssboat.example.json
ONEX_ZBANK_OFFICER_PIN=724265
ONEX_ZBANK_OFFICER_SIGNATURE=BernardGreeffNiehaus-DSSBOAT
```

---

## 5. Linked Z Bank accounts

Default linked accounts (M1–M4 operational layers):

| Account ID | Layer | Role |
|------------|-------|------|
| `zbank-usd-checking` | M1 | Spendable operating wallet |
| `zbank-usd-safeguarded` | M2 | Safeguarded settlement |
| `zbank-usd-treasury` | M3 | Treasury liquidity |
| `zbank-usd-wholesale` | M4 | Wholesale / tokenized |

---

## 6. API contract

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/bridge/bank/officer/status` | Officer store health + counts |
| GET | `/bridge/bank/officer/list` | List officers (no secrets) |
| GET | `/bridge/bank/officer?id=` | Public officer profile |
| POST | `/bridge/bank/officer/verify` | Verify PIN + signature |
| POST | `/bridge/bank/officer/transfer` | Authorized Z Bank transfer (PIN + signature required) |
| POST | `/bridge/bank/officer/ensure` | Seed DSSBOaT officer from config file |

### 6.1 Verify request

```json
{
  "officerId": "dssboat-officer-bneihaus",
  "pin": "724265",
  "signature": "BernardGreeffNiehaus-DSSBOAT"
}
```

### 6.2 Authorized transfer request

```json
{
  "officerId": "dssboat-officer-bneihaus",
  "pin": "724265",
  "signature": "BernardGreeffNiehaus-DSSBOAT",
  "fromAccount": "zbank-usd-checking",
  "toAccount": "zbank-usd-safeguarded",
  "amount": "1000.00",
  "reference": "DSSBOAT-ops-001"
}
```

---

## 7. Attachments

| File | Description |
|------|-------------|
| `docs/cis/DSSBOAT_CIS_2026.pdf` | Original Company Information Sheet + passport pages |
| `configs/zbank-officers.dssboat.example.json` | Machine-readable officer seed |

---

## 8. Verify after deploy

```bash
curl -s https://HOST/bridge/bank/officer/status | jq .
curl -s https://HOST/bridge/bank/officer/list | jq .
curl -s -X POST https://HOST/bridge/bank/officer/verify \
  -H 'Content-Type: application/json' \
  -d '{"officerId":"dssboat-officer-bneihaus","pin":"...","signature":"..."}'
```
