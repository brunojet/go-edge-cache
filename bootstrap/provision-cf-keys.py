#!/usr/bin/env python3
"""
provision-cf-keys.py

Reads a secret from AWS Secrets Manager containing a JSON with at least a `public_key` field
and provisions (idempotently) a CloudFront Public Key and a Key Group referencing it.

Usage (Git Bash / Linux):
  python3 bootstrap/provision-cf-keys.py --secret-name go-edge-cache/cf-keys --region us-east-1 \
    --public-key-name go-edge-cache-cf-pubkey --key-group-name go-edge-cache-cf-kg

Notes:
- The script is safe to run multiple times: it looks up by name and reuses existing PublicKey/KeyGroup.
- Requires AWS credentials with `secretsmanager:GetSecretValue`, `cloudfront:CreatePublicKey`,
  `cloudfront:CreateKeyGroup`, `cloudfront:GetPublicKey`, `cloudfront:ListPublicKeys`,
  `cloudfront:GetKeyGroup`, `cloudfront:ListKeyGroups`.
- Keep private key in Secrets Manager (or other vault) and let your app retrieve it at runtime.
"""

import argparse
import boto3
import botocore
import json
import time


def find_public_key_by_name(cf, name):
    # list all public keys and check their names
    paginator = cf.get_paginator('list_public_keys')
    for page in paginator.paginate():
        # paginator may return PublicKeyList as a dict with 'Items' or as a list
        pk_list = page.get('PublicKeyList')
        items = []
        if isinstance(pk_list, dict):
            items = pk_list.get('Items', [])
        elif isinstance(pk_list, list):
            items = pk_list

        for item in items:
            if isinstance(item, dict):
                pk_id = item.get('Id')
            else:
                pk_id = item

            if not pk_id:
                continue

            try:
                info = cf.get_public_key(Id=pk_id)
                cfg = info.get('PublicKey', {}).get('PublicKeyConfig', {})
                if cfg.get('Name') == name:
                    return pk_id
            except botocore.exceptions.ClientError:
                continue
    return None


def find_key_group_by_name(cf, name):
    # list_key_groups is not pageable via get_paginator in some botocore versions;
    # iterate manually using Marker/NextMarker to be robust across SDKs.
    marker = None
    while True:
        if marker:
            resp = cf.list_key_groups(Marker=marker)
        else:
            resp = cf.list_key_groups()

        kg_list = resp.get('KeyGroupList')
        items = []
        if isinstance(kg_list, dict):
            items = kg_list.get('Items', [])
        elif isinstance(kg_list, list):
            items = kg_list

        for item in items:
            if isinstance(item, dict):
                kg_id = item.get('Id')
            else:
                kg_id = item

            if not kg_id:
                continue

            try:
                info = cf.get_key_group(Id=kg_id)
                cfg = info.get('KeyGroup', {}).get('KeyGroupConfig', {})
                if cfg.get('Name') == name:
                    return kg_id
            except botocore.exceptions.ClientError:
                continue

        # handle manual pagination
        is_truncated = resp.get('IsTruncated')
        if not is_truncated:
            break
        marker = resp.get('NextMarker')

    return None


def main():
    p = argparse.ArgumentParser()
    p.add_argument('--secret-name', required=True)
    p.add_argument('--region', default='us-east-1')
    p.add_argument('--public-key-name', required=True)
    p.add_argument('--key-group-name', required=True)
    p.add_argument('--comment', default='Provisioned by script')
    args = p.parse_args()

    sm = boto3.client('secretsmanager', region_name=args.region)
    cf = boto3.client('cloudfront', region_name=args.region)

    # fetch secret
    try:
        sec = sm.get_secret_value(SecretId=args.secret_name)
    except botocore.exceptions.ClientError as e:
        print('ERROR: unable to read secret:', e)
        raise SystemExit(1)

    secret_string = sec.get('SecretString')
    if not secret_string:
        print('ERROR: secret has no SecretString')
        raise SystemExit(2)

    # If secret is JSON, support {"public_key": "...", "private_key": "..."}
    try:
        secret_json = json.loads(secret_string)
        public_key_pem = secret_json.get('public_key') or secret_json.get('public')
    except Exception:
        # secret is raw PEM
        public_key_pem = secret_string

    if not public_key_pem:
        print('ERROR: public_key not found in secret')
        raise SystemExit(3)

    # find or create public key
    pk_id = find_public_key_by_name(cf, args.public_key_name)
    if pk_id:
        print('Found existing CloudFront PublicKey id=', pk_id)
    else:
        print('Creating CloudFront PublicKey...')
        caller_ref = str(int(time.time() * 1000))
        cfg = {
            'CallerReference': caller_ref,
            'Name': args.public_key_name,
            'EncodedKey': public_key_pem,
            'Comment': args.comment,
        }
        try:
            resp = cf.create_public_key(PublicKeyConfig=cfg)
            pk_id = resp['PublicKey']['Id']
            print('Created PublicKey id=', pk_id)
        except botocore.exceptions.ClientError as e:
            code = e.response.get('Error', {}).get('Code', '')
            if code == 'PublicKeyAlreadyExists':
                print('PublicKeyAlreadyExists: locating existing key by name...')
                pk_id = find_public_key_by_name(cf, args.public_key_name)
                if pk_id:
                    print('Found existing PublicKey id=', pk_id)
                else:
                    print('ERROR: PublicKeyAlreadyExists but could not locate existing key id')
                    raise
            else:
                raise

    # find or create key group
    kg_id = find_key_group_by_name(cf, args.key_group_name)
    if kg_id:
        print('Found existing KeyGroup id=', kg_id)
    else:
        print('Creating KeyGroup referencing PublicKey id=', pk_id)
        kg_cfg = {
            'Name': args.key_group_name,
            'Items': [pk_id],
            'Comment': args.comment,
        }
        try:
            resp = cf.create_key_group(KeyGroupConfig=kg_cfg)
            kg_id = resp['KeyGroup']['Id']
            print('Created KeyGroup id=', kg_id)
        except botocore.exceptions.ClientError as e:
            code = e.response.get('Error', {}).get('Code', '')
            if code == 'KeyGroupAlreadyExists':
                print('KeyGroupAlreadyExists: locating existing key group by name...')
                kg_id = find_key_group_by_name(cf, args.key_group_name)
                if kg_id:
                    print('Found existing KeyGroup id=', kg_id)
                else:
                    print('ERROR: KeyGroupAlreadyExists but could not locate existing key group id')
                    raise
            else:
                raise

    print('\nDone. PublicKeyId=', pk_id, ' KeyGroupId=', kg_id)


if __name__ == '__main__':
    main()
