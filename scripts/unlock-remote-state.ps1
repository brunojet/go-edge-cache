param(
    [string]$Bucket = "brunojet-tfstate",
    [string]$Key = "go-edge-cache/terraform.tfstate",
    [string]$Region = "us-east-1"
)

$lockKey = "$Key.lock"
Write-Host "Removing lock s3://$Bucket/$lockKey"
aws s3 rm "s3://$Bucket/$lockKey" --region $Region | Out-Null
Write-Host "Unlocked."
