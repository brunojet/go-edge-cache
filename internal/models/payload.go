package models

// SecretPayload matches go-edge-key-management structure.
// Used for CloudFront signed URL key material.
type SecretPayload struct {
	PrivatePEM   string `json:"private_pem"`
	PublicPEM    string `json:"public_pem"`
	Fingerprint  string `json:"fingerprint"`
	CreatedAt    string `json:"created_at"`
	KeyGroupName string `json:"key_group_name"`
	NamePrefix   string `json:"name_prefix"`
	PublicKeyID  string `json:"public_key_id"`
}
