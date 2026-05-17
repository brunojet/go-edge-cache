param(
    [string]$Bucket = "brunojet-tfstate",
    [string]$Key = "go-edge-cache/terraform.tfstate",
    [string]$Region = "us-east-1"
)

$tmpFile = [System.IO.Path]::GetTempFileName()
$lockFile = [System.IO.Path]::GetTempFileName()

$uuid = [guid]::NewGuid().ToString()

$state = @{ 
    version = 4
    terraform_version = "1.5.0"
    serial = 1
    lineage = $uuid
    resources = @()
}

$json = $state | ConvertTo-Json -Depth 10
$json | Out-File -FilePath $tmpFile -Encoding UTF8

"locked by $env:USERNAME at $(Get-Date -Format o)" | Out-File -FilePath $lockFile -Encoding UTF8

Write-Host "Uploading state to s3://$Bucket/$Key (region=$Region)"
aws s3 cp $tmpFile "s3://$Bucket/$Key" --region $Region

Write-Host "Creating lock file s3://$Bucket/$Key.lock"
aws s3 cp $lockFile "s3://$Bucket/$Key.lock" --region $Region

Write-Host "Done. To remove the lock run: .\scripts\unlock-remote-state.ps1 -Bucket $Bucket -Key $Key -Region $Region"
