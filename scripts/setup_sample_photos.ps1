# Setup Sample Dog Photos for Testing
# Copies sample SVG photos to uploads directory for testing

param(
    [string]$UploadDir = ".\uploads\dogs"
)

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "  Sample Dog Photos Setup" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

# Create uploads directory if it doesn't exist
if (-not (Test-Path $UploadDir)) {
    Write-Host "Creating uploads directory: $UploadDir" -ForegroundColor Yellow
    New-Item -ItemType Directory -Path $UploadDir -Force | Out-Null
}

# Check if sample photos exist
$samplePhotos = @(
    @{Source="scripts\sample_photos\dog_sample_1.svg"; Dest="dog_1_full.jpg"; Thumb="dog_1_thumb.jpg"},
    @{Source="scripts\sample_photos\dog_sample_2.svg"; Dest="dog_2_full.jpg"; Thumb="dog_2_thumb.jpg"},
    @{Source="scripts\sample_photos\dog_sample_3.svg"; Dest="dog_3_full.jpg"; Thumb="dog_3_thumb.jpg"}
)

$copiedCount = 0

foreach ($photo in $samplePhotos) {
    if (Test-Path $photo.Source) {
        # Copy as full-size photo
        $destFull = Join-Path $UploadDir $photo.Dest
        Copy-Item -Path $photo.Source -Destination $destFull -Force
        Write-Host "[OK] Copied $($photo.Dest)" -ForegroundColor Green

        # Copy as thumbnail (same file for SVG testing)
        $destThumb = Join-Path $UploadDir $photo.Thumb
        Copy-Item -Path $photo.Source -Destination $destThumb -Force
        Write-Host "[OK] Copied $($photo.Thumb)" -ForegroundColor Green

        $copiedCount++
    } else {
        Write-Host "[SKIP] Source not found: $($photo.Source)" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "======================================" -ForegroundColor Green
Write-Host "  Summary" -ForegroundColor Green
Write-Host "======================================" -ForegroundColor Green
Write-Host ""
Write-Host "  Copied: $copiedCount sample photos" -ForegroundColor Green
Write-Host "  Location: $UploadDir" -ForegroundColor Cyan
Write-Host ""
Write-Host "Sample photos are now available for testing!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "  1. Run test data generator: .\scripts\gentestdata.ps1" -ForegroundColor White
Write-Host "  2. Update database to reference these photos (manual SQL or via upload UI)" -ForegroundColor White
Write-Host "  3. Start application: go run cmd/server/main.go" -ForegroundColor White
Write-Host "  4. View dogs with photos at: http://localhost:8080/dogs.html" -ForegroundColor White
Write-Host ""
