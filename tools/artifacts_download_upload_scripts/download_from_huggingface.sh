#!/bin/bash
# Script to download model artifacts from Hugging Face

CONFIG_FILE="./model_artifacts_configs.yaml"
DOWNLOAD_DIR="./model_artifacts"

# Ensure download directory exists
mkdir -p "$DOWNLOAD_DIR"

if ! command -v wget &> /dev/null; then
    echo "wget not found, installing..."
    if [[ "$(uname -s)" == "Darwin" ]]; then
        if command -v brew &> /dev/null; then
            brew install wget
        else
            echo "Error: Homebrew not found. Please install wget manually or install Homebrew first."
            exit 1
        fi
    else
        if [ "$(id -u)" -eq 0 ]; then
            apt-get update && apt-get install -y wget
        elif command -v sudo &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y wget
        else
            echo "Error: Root privileges are required to install wget. Please run this script as root or install wget manually."
            exit 1
        fi
    fi
fi

# Find the correct yq 
YQ_CMD=""
if command -v brew &> /dev/null && [[ -f "$(brew --prefix yq 2>/dev/null)/bin/yq" ]]; then
    YQ_CMD="$(brew --prefix yq)/bin/yq"
    echo "Using Homebrew yq: $YQ_CMD"
elif command -v yq &> /dev/null && yq --version 2>&1 | grep -q "mikefarah"; then
    YQ_CMD="yq"
else
    echo "yq (mikefarah's version) not found, installing..."
    OS="$(uname -s)"
    ARCH="$(uname -m)"
    case "$OS" in
        Linux*)
            if [[ "$ARCH" == "x86_64" ]]; then
                YQ_BINARY="yq_linux_amd64"
            elif [[ "$ARCH" == "aarch64" || "$ARCH" == "arm64" ]]; then
                YQ_BINARY="yq_linux_arm64"
            else
                echo "Unsupported architecture: $ARCH"
                exit 1
            fi
            ;;
        Darwin*)
            if [[ "$ARCH" == "x86_64" ]]; then
                YQ_BINARY="yq_darwin_amd64"
            elif [[ "$ARCH" == "arm64" ]]; then
                YQ_BINARY="yq_darwin_arm64"
            else
                echo "Unsupported architecture: $ARCH"
                exit 1
            fi
            ;;
        *)
            echo "Unsupported OS: $OS"
            exit 1
            ;;
    esac
    wget "https://github.com/mikefarah/yq/releases/download/v4.44.1/$YQ_BINARY" -O /usr/local/bin/yq
    chmod +x /usr/local/bin/yq
    YQ_CMD="/usr/local/bin/yq"
fi

# HF_TOKEN and HF_USERNAME are set in the model_artifacts_configs.yaml file
HF_TOKEN=$("$YQ_CMD" -r '.hf-token' "$CONFIG_FILE")
HF_USERNAME=$("$YQ_CMD" -r '.hf-username' "$CONFIG_FILE")
echo "HF_TOKEN: $HF_TOKEN"
echo "HF_USERNAME: $HF_USERNAME"

if ! command -v git-lfs &> /dev/null; then
    echo "git-lfs not found, installing..."
    if [[ "$(uname -s)" == "Darwin" ]]; then
        if command -v brew &> /dev/null; then
            brew install git-lfs
        else
            echo "Error: Homebrew not found. Please install git-lfs manually or install Homebrew first."
            exit 1
        fi
    else
        apt-get update && apt-get install -y git-lfs
    fi
    git lfs install
fi

if [ -f "$CONFIG_FILE" ]; then
    echo "Reading $CONFIG_FILE"
    
    # Get total count of artifacts
    artifact_count=$("$YQ_CMD" '.artifact-configs | length' "$CONFIG_FILE")
    echo "Found $artifact_count artifacts to download"
    echo ""
    
    # Process all artifacts in the config
    for ((idx=0; idx<artifact_count; idx++)); do
        id=$("$YQ_CMD" -r ".artifact-configs[$idx].artifact-id" "$CONFIG_FILE")
        echo "Processing artifact ID: $id"
        
        # Get artifact configuration
        hf_url=$("$YQ_CMD" -r ".artifact-configs[$idx].hf-url" "$CONFIG_FILE")
        files_to_exclude=$("$YQ_CMD" -r ".artifact-configs[$idx].files-to-exclude[]?" "$CONFIG_FILE")
        is_a_gated_model=$("$YQ_CMD" -r ".artifact-configs[$idx].is-a-gated-model" "$CONFIG_FILE")
        
        echo "hf-url: $hf_url"
        echo "files-to-exclude: $files_to_exclude"
        echo "is-a-gated-model: $is_a_gated_model"
        
        if [[ -n "$hf_url" && "$hf_url" != "null" ]]; then
            # Clone from Hugging Face
            if [[ "$is_a_gated_model" == "true" ]]; then
                HF_USERNAME_ENC=$(python3 -c "import urllib.parse; print(urllib.parse.quote('''$HF_USERNAME'''))")
                auth_hf_url=$(echo "$hf_url" | sed "s#https://#https://$HF_USERNAME_ENC:$HF_TOKEN@#")
                echo "Cloning gated model $hf_url for $id"
                git clone "$auth_hf_url" "$DOWNLOAD_DIR/$id"
            else
                echo "Cloning $hf_url for $id"
                git clone "$hf_url" "$DOWNLOAD_DIR/$id"
            fi
            
            # Clean up git files
            find "$DOWNLOAD_DIR/$id" -type f \( -name ".gitattributes" -o -name ".gitignore" -o -name ".gitmodules" \) -exec rm -f {} +
            rm -rf "$DOWNLOAD_DIR/$id/.git"
            
            # Exclude files-to-exclude
            if [[ -n "$files_to_exclude" ]]; then
                shopt -s nullglob
                while IFS= read -r exclude_file; do
                    if [[ "$exclude_file" == */ ]]; then
                        rm -rf "$DOWNLOAD_DIR/$id/$exclude_file"
                        echo "Excluded folder $exclude_file"
                    elif [[ "$exclude_file" == *"*"* || "$exclude_file" == *"?"* ]]; then
                        for match in "$DOWNLOAD_DIR/$id"/$exclude_file; do
                            if [ -e "$match" ]; then
                                rm -f "$match"
                                echo "Excluded $match"
                            fi
                        done
                    else
                        rm -f "$DOWNLOAD_DIR/$id/$exclude_file"
                        echo "Excluded $exclude_file"
                    fi
                done <<< "$files_to_exclude"
                shopt -u nullglob
            fi
            
            ls -lR "$DOWNLOAD_DIR/$id"
            echo "Successfully downloaded $id to $DOWNLOAD_DIR/$id"
        else
            echo "hf-url not set for $id, skipping clone."
        fi
        
        echo "-----------------------------"
    done
    
    echo ""
    echo "Download complete! Artifacts are located in: $DOWNLOAD_DIR"
    echo "To upload to MinIO, run: ./upload_to_minio.sh"
else
    echo "$CONFIG_FILE not found!"
    exit 1
fi

