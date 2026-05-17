Signed URLs: recommended workflow

Goal: avoid embedding keys in Terraform variables or state. Store keys securely and provision CloudFront public key + key group from the public key.

Recommended pattern (safe):

1) Store keys in AWS Secrets Manager (out of Terraform):
   - Secret name: e.g. `go-edge-cache/cf-keys`
   - SecretString JSON: e.g.
     {
       "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
       "public_key": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"
     }

   Example CLI (Git Bash):

   ```bash
   aws secretsmanager create-secret --name go-edge-cache/cf-keys \
     --description "CF signed URL key pair" \
     --secret-string file://terraform/keys/cf-keys.json --region us-east-1
   ```

2) Provision CloudFront Public Key + Key Group from Secrets Manager (outside Terraform)
   - Use `scripts/provision-cf-keys.py` to read the secret and create the CloudFront resources idempotently.

   Example:
   ```bash
   python3 scripts/provision-cf-keys.py \
     --secret-name go-edge-cache/cf-keys \
     --region us-east-1 \
     --public-key-name go-edge-cache-cf-pubkey \
     --key-group-name go-edge-cache-cf-kg
   ```

   The script will print the `PublicKeyId` and `KeyGroupId` which you can then reference in Terraform (if desired) as a variable `signed_key_group_id`.

3) Application usage (runtime):
   - Application retrieves the `private_key` from Secrets Manager via the AWS SDK and uses it to sign URLs.
   - Do NOT store the private key in Git, Terraform state, or logs.

4) Optional Terraform integration:
   - If you accept that the public key id (not the private key) may be referenced in Terraform, you can pass the `KeyGroupId` as a variable to the module instead of creating it inside Terraform.
   - This keeps secrets out of the state while allowing Terraform to reference the created KeyGroup by id.

Security notes:
- Prefer Secrets Manager (rotation, fine-grained IAM) or SSM SecureString. Avoid using plain S3 for secrets.
- Never put the private key into Terraform-managed resources that will be stored in State.

