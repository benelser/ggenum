# ggenum
ggenum (google groups enumeration) looks for misconfigured groups that could be abused for escalation of privs. Inspired by [Hacking G Suite: The Power
of Dark Apps Script Magic](029%20presentations/Matthew%20Bryant%20-%20Hacking%20G%20Suite%20-%20%20The%20Power%20of%20Dark%20Apps%20Script%20Magic.pdf) By @IAmMandatory (Matthew Bryant)

## Steps to run
1. Create gcp project
2. Configure consent screen
3. Enable apis
```bash
groupssettings.googleapis.com
admin.googleapis.com
```
4. Create Web App OAuth Cred
5. Download key and rename to key.json
6. Execute
```bash
go run main.go -customer_id=CUSTOMER_ID --key=OAUTH_CLIENT_CREDS.json
```
7. Copy paste login url in browser and sign in using creds that have access. (Subsequent runs reuses valid token)# ggenum
