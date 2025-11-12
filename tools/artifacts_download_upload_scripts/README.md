# Model Artifacts Scripts

This directory contains scripts for downloading model artifacts from Hugging Face and uploading them to MinIO/S3.

## Scripts

### 1. `download_from_huggingface.sh`
Downloads model artifacts from Hugging Face repositories.

**Features:**
- Reads configuration from `model_artifacts_configs.yaml`
- Supports both public and gated Hugging Face models
- Automatically installs dependencies (wget, yq, git-lfs)
- Cleans up git files after download
- Excludes specified files based on configuration
- Saves downloads to `./model_artifacts/` directory

**Usage:**
```bash
./download_from_huggingface.sh
```

**Prerequisites:**
- `model_artifacts_configs.yaml` must be present in the same directory
- For gated models: HF token and username must be configured in the YAML file

### 2. `upload_to_minio.sh`
Uploads downloaded artifacts to MinIO storage.

**Features:**
- Automatically uploads **all artifacts** from `./model_artifacts/` directory
- No config file needed - just uploads everything found
- **Auto-creates bucket** if it doesn't exist
- Uses native MinIO Client (mc) for optimal performance
- Comprehensive dependency installation:
  - MinIO Client (via Homebrew or direct download)
  - Supports macOS (Intel & Apple Silicon) and Linux (amd64 & arm64)
  - Multiple fallback installation methods

**Usage:**
```bash
./upload_to_minio.sh
```

**Prerequisites:**
- Run `download_from_huggingface.sh` first to download artifacts
- Configure MinIO settings in the script or use environment variables:
  - `MINIO_ENDPOINT` (default: http://127.0.0.1:9000)
  - `MINIO_BUCKET` (default: personal)
  - `MINIO_ROOT_USER` (default: minioadmin)
  - `MINIO_ROOT_PASSWORD` (default: minioadmin)

### 3. `upload_to_minio_aws.sh`
Uploads downloaded artifacts to MinIO using AWS CLI (S3-compatible API).

**Features:**
- Automatically uploads **all artifacts** from `./model_artifacts/` directory
- No config file needed - just uploads everything found
- **Auto-creates bucket** if it doesn't exist
- Uses AWS CLI with S3-compatible API for MinIO
- Comprehensive dependency installation:
  - AWS CLI (via Homebrew or official installer)
  - Supports macOS (Intel & Apple Silicon) and Linux (amd64 & arm64)
  - Multiple fallback installation methods
- Alternative to `upload_to_minio.sh` (uses AWS CLI instead of mc)

**Usage:**
```bash
./upload_to_minio_aws.sh
```

**Prerequisites:**
- Run `download_from_huggingface.sh` first to download artifacts
- Configure MinIO settings in the script:
  - `MINIO_ENDPOINT` (default: http://127.0.0.1:9000)
  - `MINIO_BUCKET` (default: ml-platform-artifacts)
  - `MINIO_ACCESS_KEY` (default: minioadmin)
  - `MINIO_SECRET_KEY` (default: minioadmin)

**When to use this vs `upload_to_minio.sh`:**
- Use this if you prefer AWS CLI over MinIO Client (mc)
- Use this if you already have AWS CLI installed
- Use `upload_to_minio.sh` for better MinIO native support

### 4. `upload_to_s3.sh`
Uploads downloaded artifacts to AWS S3 storage.

**Features:**
- Automatically uploads **all artifacts** from `./model_artifacts/` directory
- No config file needed - just uploads everything found
- **Auto-creates bucket** if it doesn't exist (with proper region configuration)
- Uses AWS CLI with proper credential validation
- Comprehensive dependency installation:
  - AWS CLI (via Homebrew or official installer)
  - Supports macOS (Intel & Apple Silicon) and Linux (amd64 & arm64)
  - Multiple fallback installation methods
- Validates AWS credentials before upload

**Usage:**
```bash
export S3_BUCKET=your-bucket-name
export S3_REGION=us-east-1  # Optional, defaults to us-east-1
export S3_PREFIX=model_artifacts  # Optional, defaults to 'model_artifacts'
./upload_to_s3.sh
```

Or set inline:
```bash
S3_BUCKET=your-bucket-name S3_REGION=us-west-2 ./upload_to_s3.sh
```

**Prerequisites:**
- Run `download_from_huggingface.sh` first to download artifacts
- AWS credentials must be configured:
  - AWS CLI configuration (`aws configure`)
  - Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
  - IAM role (if running on AWS infrastructure)
- Set `S3_BUCKET` environment variable
- Optional: Set `S3_REGION` (default: us-east-1) and `S3_PREFIX` (default: model_artifacts)

### 5. `test_minio_connection.sh`
Diagnostic script to test MinIO connectivity and troubleshoot issues.

**Features:**
- Tests MinIO Client (mc) installation
- Verifies MinIO endpoint connectivity
- Tests authentication with credentials
- Lists all existing buckets
- Tests bucket creation permissions
- Provides detailed troubleshooting information

**Usage:**
```bash
./test_minio_connection.sh
```

Or with custom settings:
```bash
MINIO_ENDPOINT=http://localhost:9000 MINIO_BUCKET=nexus ./test_minio_connection.sh
```

**When to use:**
- Before running upload scripts for the first time
- When bucket creation fails
- To diagnose MinIO connectivity issues
- To verify credentials and permissions

## Configuration

The download script uses the `model_artifacts_configs.yaml` configuration file.

### ⚠️ What You Need to Change:

**Only update these fields if you're downloading gated models:**
- `hf-token`: Your Hugging Face API token
  - Get your token from: https://huggingface.co/settings/tokens
  - **Leave as-is for public models**
- `hf-username`: Your Hugging Face username
  - **Only required for gated models**

**✅ Everything else is pre-configured - no changes needed!**

---

### Configuration File Reference (Pre-configured):

**Top-Level Fields:**
- `hf-token`: Hugging Face API token (update only for gated models)
- `hf-username`: Hugging Face username (update only for gated models)

**Artifact Configuration (`artifact-configs`):**

The following 10 models are pre-configured and ready to download:
- `all-minilm-l6-v2` - Sentence transformer model
- `bi-encoder` - BGE small encoder
- `cross-encoder` - MS MARCO cross-encoder
- `e5-language-classifier` - Multilingual language detection
- `llama31-70b-instruct-awq` - Llama 3.1 70B quantized
- `mbart-translator` - Multilingual translation
- `llama31-8b-instruct` - Llama 3.1 8B
- `pii-classifier` - PII detection model
- `uae-large` - UAE embedding model
- `xlm-roberta-language-classifier` - Language classifier

Each artifact includes:
- `artifact-id`: Unique identifier (used as directory/file name)
- `hf-url`: Hugging Face repository URL
- `is-a-gated-model`: Authentication requirement (`true`/`false`)
- `files-to-exclude`: (Optional) Files/patterns to skip during download

**Note:** 
- All artifacts listed in `artifact-configs` will be downloaded by the download script
- The upload script automatically uploads all directories found in `./model_artifacts/` - no config needed!

### Example Configuration Structure:

```yaml
hf-token: "your_hf_token_here"
hf-username: "your_username"

artifact-configs:
  - artifact-id: model-1
    hf-url: "https://huggingface.co/org/model-name"
    is-a-gated-model: false
    files-to-exclude:
      - "*.bin"
      - "test/"
  
  - artifact-id: model-2
    hf-url: "https://huggingface.co/org/gated-model"
    is-a-gated-model: true
  
  - artifact-id: model-3
    hf-url: "https://huggingface.co/org/another-model"
    is-a-gated-model: false
```

All artifacts in the list will be downloaded and uploaded automatically.

## Workflow

1. **Download artifacts from Hugging Face:**
   ```bash
   ./download_from_huggingface.sh
   ```
   This will download all configured artifacts to `./model_artifacts/` directory.

2. **Upload to storage** (choose one or more):

   **Option A - Upload to MinIO (using MinIO Client):**
   ```bash
   ./upload_to_minio.sh
   ```
   
   **Option B - Upload to MinIO (using AWS CLI):**
   ```bash
   ./upload_to_minio_aws.sh
   ```
   
   **Option C - Upload to AWS S3:**
   ```bash
   export S3_BUCKET=your-bucket-name
   ./upload_to_s3.sh
   ```
   
   You can run multiple scripts to upload to different destinations!

## Environment Variables

### For Download Script:
- No additional environment variables needed (reads from `model_artifacts_configs.yaml`)

### For MinIO Upload Script (using mc):
- No config file needed - automatically uploads all artifacts from `./model_artifacts/`
- `MINIO_ENDPOINT`: MinIO server endpoint (default: http://127.0.0.1:9000)
- `MINIO_BUCKET`: Target bucket name (default: personal)
- `MINIO_ROOT_USER`: MinIO access key (default: minioadmin)
- `MINIO_ROOT_PASSWORD`: MinIO secret key (default: minioadmin)

### For MinIO Upload Script (using AWS CLI):
- No config file needed - automatically uploads all artifacts from `./model_artifacts/`
- `MINIO_ENDPOINT`: MinIO server endpoint (default: http://127.0.0.1:9000)
- `MINIO_BUCKET`: Target bucket name (default: ml-platform-artifacts)
- `MINIO_ACCESS_KEY`: MinIO access key (default: minioadmin)
- `MINIO_SECRET_KEY`: MinIO secret key (default: minioadmin)

### For S3 Upload Script:
- No config file needed - automatically uploads all artifacts from `./model_artifacts/`
- `S3_BUCKET`: (Required) Target S3 bucket name
- `S3_REGION`: AWS region (default: us-east-1)
- `S3_PREFIX`: Path prefix in bucket (default: model_artifacts)
- AWS credentials via:
  - `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`
  - AWS CLI configuration (`~/.aws/credentials`)
  - IAM role (for EC2/ECS/Lambda)

## Notes

- The download script creates a `./model_artifacts/` directory and downloads artifacts based on `model_artifacts_configs.yaml`
- All upload scripts are config-free - they simply upload **everything** found in `./model_artifacts/` directory
- **Buckets are automatically created** if they don't exist:
  - MinIO: Creates bucket using `mc mb` command
  - S3: Creates bucket with appropriate region configuration
- **Bucket names are automatically normalized** to lowercase:
  - MinIO/S3 require lowercase bucket names
  - Scripts automatically convert names like "Nexus" to "nexus"
  - Warning displayed if bucket name contains invalid characters
- This means you can manually place any additional artifacts in `./model_artifacts/` and they will be uploaded
- You can upload to both MinIO and S3 if needed - just run both upload scripts
- All scripts support macOS (Darwin) and Linux environments
- Dependencies are automatically installed if missing:
  - **Download script**: wget, yq, git-lfs
  - **MinIO upload script (mc)**: MinIO Client (mc) - native client for MinIO
  - **MinIO upload script (AWS CLI)**: AWS CLI - S3-compatible API for MinIO
  - **S3 upload script**: AWS CLI - official AWS command line tool
- Architecture support: Intel/AMD64 and ARM64 (Apple Silicon, AWS Graviton, etc.)
- The original combined script is retained for backwards compatibility

## Dependency Installation Methods

### Download Script Dependencies:
- wget, yq, git-lfs (automatically installed based on OS)

### MinIO Upload Script Dependencies:
Installs MinIO Client (mc):

1. **macOS**:
   - Homebrew (if installed): `brew install minio/stable/mc`
   - Direct download: Downloads appropriate binary (Intel or Apple Silicon)
   - Installs to `/usr/local/bin/mc`

2. **Linux**:
   - Direct download: Downloads appropriate binary (amd64 or arm64)
   - Installs to `/usr/local/bin/mc` (with sudo) or `~/.local/bin/mc` (without sudo)
   - Provides manual installation instructions if all methods fail

### S3 Upload Script Dependencies:
Installs AWS CLI:

1. **macOS**:
   - Homebrew (if installed): `brew install awscli`
   - Official installer: Downloads and installs AWSCLIV2.pkg

2. **Linux**:
   - Official installer: Downloads appropriate binary (amd64 or arm64)
   - Installs to `/usr/local/aws-cli` (with sudo) or `~/.local/aws-cli` (without sudo)
   - Requires unzip utility (auto-installed if missing)

