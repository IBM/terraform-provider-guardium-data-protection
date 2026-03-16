#!/bin/bash

set -e

# IBM Artifactory Configuration
ARTIFACTORY_URL="https://na.artifactory.swg-devops.com"
ARTIFACTORY_REPO="sec-guardium-next-gen-terraform-local"
ARTIFACTORY_TOKEN="${ARTIFACTORY_TOKEN}"

# Provider Configuration
PROVIDER_NAMESPACE="registry.terraform.io/ibm/guardium-data-protection"
VERSION="${1:-1.0.0}"

if [ -z "$ARTIFACTORY_TOKEN" ]; then
  echo "Error: ARTIFACTORY_TOKEN environment variable not set"
  echo "Get your token from: https://na.artifactory.swg-devops.com/ui/admin/artifactory/user_profile"
  exit 1
fi

echo "=========================================="
echo "Uploading Guardium Data Protection Provider to IBM Artifactory"
echo "=========================================="
echo "Repository: ${ARTIFACTORY_REPO}"
echo "Provider: ${PROVIDER_NAMESPACE}"
echo "Version: ${VERSION}"
echo ""

# Build all platforms
echo "Building provider for all platforms..."
make build-all
echo ""

# Upload each platform binary as zip (Terraform network mirror requires zip archives)
PLATFORMS=("linux_amd64" "darwin_arm64" "darwin_amd64" "windows_amd64")
ARCHIVES_JSON=""

for platform in "${PLATFORMS[@]}"; do
  echo "Packaging and uploading ${platform}..."

  # Find the binary in dist directory (goreleaser creates subdirectories)
  if [ "$platform" = "windows_amd64" ]; then
    BINARY_PATH=$(find dist -name "terraform-provider-guardium-data-protection*" -path "*${platform}*" -name "*.exe" -type f | head -1)
    BINARY_NAME="terraform-provider-guardium-data-protection.exe"
  else
    BINARY_PATH=$(find dist -name "terraform-provider-guardium-data-protection*" -path "*${platform}*" ! -name "*.exe" -type f | head -1)
    BINARY_NAME="terraform-provider-guardium-data-protection"
  fi

  if [ -z "$BINARY_PATH" ]; then
    echo "Warning: Binary not found for ${platform}, skipping..."
    continue
  fi

  echo "  Found binary: ${BINARY_PATH}"

  # Create zip archive (Terraform requires zip format)
  ZIP_NAME="terraform-provider-guardium-data-protection_${VERSION}_${platform}.zip"
  rm -f "${ZIP_NAME}"
  
  # Create a temporary directory to ensure correct filename in zip
  TEMP_DIR=$(mktemp -d)
  cp "${BINARY_PATH}" "${TEMP_DIR}/${BINARY_NAME}"
  (cd "${TEMP_DIR}" && zip -q "../${ZIP_NAME}" "${BINARY_NAME}")
  rm -rf "${TEMP_DIR}"

  # Compute SHA256 hash
  ZIP_HASH=$(shasum -a 256 "${ZIP_NAME}" | awk '{print $1}')

  # Target path in Artifactory
  TARGET_PATH="artifactory/${ARTIFACTORY_REPO}/${PROVIDER_NAMESPACE}/${VERSION}/${ZIP_NAME}"

  # Upload zip using Artifactory API
  HTTP_CODE=$(curl -s -w "%{http_code}" -o /tmp/upload_response.txt \
       -H "X-JFrog-Art-Api: ${ARTIFACTORY_TOKEN}" \
       -X PUT \
       -T "${ZIP_NAME}" \
       "${ARTIFACTORY_URL}/${TARGET_PATH}")

  if [ "$HTTP_CODE" -eq 201 ] || [ "$HTTP_CODE" -eq 200 ]; then
    echo "✓ Uploaded ${platform} (HTTP ${HTTP_CODE})"
  else
    echo "✗ Failed to upload ${platform} (HTTP ${HTTP_CODE})"
    echo "Response:"
    cat /tmp/upload_response.txt
    echo ""
    exit 1
  fi

  # Build archives JSON entry
  if [ -n "$ARCHIVES_JSON" ]; then
    ARCHIVES_JSON="${ARCHIVES_JSON},"
  fi
  ARCHIVES_JSON="${ARCHIVES_JSON}
    \"${platform}\": {
      \"url\": \"${VERSION}/${ZIP_NAME}\",
      \"hashes\": [\"zh:${ZIP_HASH}\"]
    }"

  rm -f "${ZIP_NAME}"
done

echo ""
echo "Uploading metadata files..."

# Upload version JSON (1.0.0.json)
VERSION_JSON="{
  \"archives\": {${ARCHIVES_JSON}
  }
}"
echo "${VERSION_JSON}" > version.json

HTTP_CODE=$(curl -s -w "%{http_code}" -o /tmp/upload_response.txt \
     -H "X-JFrog-Art-Api: ${ARTIFACTORY_TOKEN}" \
     -H "Content-Type: application/json" \
     -X PUT \
     -T version.json \
     "${ARTIFACTORY_URL}/artifactory/${ARTIFACTORY_REPO}/${PROVIDER_NAMESPACE}/${VERSION}.json")

if [ "$HTTP_CODE" -eq 201 ] || [ "$HTTP_CODE" -eq 200 ]; then
  echo "✓ Uploaded ${VERSION}.json (HTTP ${HTTP_CODE})"
else
  echo "✗ Failed to upload ${VERSION}.json (HTTP ${HTTP_CODE})"
  cat /tmp/upload_response.txt
  exit 1
fi

# Upload index JSON
INDEX_JSON='{
  "versions": {
    "'"${VERSION}"'": {}
  }
}'
echo "${INDEX_JSON}" > index.json

HTTP_CODE=$(curl -s -w "%{http_code}" -o /tmp/upload_response.txt \
     -H "X-JFrog-Art-Api: ${ARTIFACTORY_TOKEN}" \
     -H "Content-Type: application/json" \
     -X PUT \
     -T index.json \
     "${ARTIFACTORY_URL}/artifactory/${ARTIFACTORY_REPO}/${PROVIDER_NAMESPACE}/index.json")

if [ "$HTTP_CODE" -eq 201 ] || [ "$HTTP_CODE" -eq 200 ]; then
  echo "✓ Uploaded index.json (HTTP ${HTTP_CODE})"
else
  echo "✗ Failed to upload index.json (HTTP ${HTTP_CODE})"
  cat /tmp/upload_response.txt
  exit 1
fi

# Cleanup
rm -f version.json index.json

echo ""
echo "=========================================="
echo "✓ Upload Complete!"
echo "=========================================="
echo ""
echo "Provider uploaded to:"
echo "  ${ARTIFACTORY_URL}/ui/repos/tree/General/${ARTIFACTORY_REPO}/${PROVIDER_NAMESPACE}/${VERSION}"
echo ""
echo "Team members should configure Terraform with:"
echo ""
echo "  ~/.netrc:"
echo "    machine na.artifactory.swg-devops.com"
echo "    login your-email@ibm.com"
echo "    password your-api-token"
echo ""
echo "  ~/.terraformrc:"
echo "    provider_installation {"
echo "      network_mirror {"
echo "        url = \"${ARTIFACTORY_URL}/artifactory/${ARTIFACTORY_REPO}/\""
echo "        include = [\"ibm/*\"]"
echo "      }"
echo "      direct {"
echo "        exclude = [\"ibm/*\"]"
echo "      }"
echo "    }"
echo ""
echo "Then in your Terraform modules use:"
echo "  terraform {"
echo "    required_providers {"
echo "      guardium-data-protection = {"
echo "        source  = \"registry.terraform.io/ibm/guardium-data-protection\""
echo "        version = \"${VERSION}\""
echo "      }"
echo "    }"
echo "  }"
echo ""
echo "Then run: terraform init"
echo "=========================================="

# Made with Bob
