# CFS3

CFS3 is a hacky, S3-like file management tool built on top of **Cloudflare Pages**. It allows you to manage files (upload, update, remove) incrementally without needing to re-upload your entire site every time.

It leverages **Cloudflare D1** to maintain the state of your file system and **Cloudflare Pages** for hosting the content.

## 🚀 Features

- **Incremental Updates**: Only upload changed files.
- **S3-like Experience**: "Patch" (add/update) or "Remove" files.
- **Metadata Tracking**: Uses Cloudflare D1 to track file state.
- **Custom Headers**: Easily configure response headers.
- **Deduplication**: Automatically detects and skips uploading existing files.

## 🛠️ Prerequisites

You need the following Cloudflare resources:
1.  **Cloudflare Account**
2.  **Cloudflare Pages Project**
3.  **Cloudflare D1 Database**

## ⚙️ Configuration

Set the following environment variables:

```bash
export CF_ACCOUNT_ID="your_account_id"
export CF_API_TOKEN="your_api_token"
export CF_DATABASE_ID="your_d1_database_id"
```

Create a `cfs3.config.json` file in your project root:

```json
{
  "by": "user-id",
  "mode": "patch",
  "project_name": "my-pages-project",
  "headers": {
    "Access-Control-Allow-Origin": "*"
  },
  "files__patch": [
    {
      "local_file": "./local/path/image.png",
      "remote_dir": "/images",
      "metadata": { "foo": "bar" }
    }
  ]
}
```

### Modes

- **`patch`**: Adds or updates files.
- **`remove`**: Removes files (requires `files__remove` list of IDs).

## 📦 Usage

Run the tool to apply your configuration:

```bash
go run app/main.go [config_file]
```

*If no config file is specified, it defaults to `cfs3.config.json`.*

## 🧠 How it Works

1.  **State Management**: CFS3 connects to your D1 database to fetch the current state of your files.
2.  **Diffing**: It calculates hashes of your local files and checks against Cloudflare Pages to see what actually needs to be uploaded.
3.  **Deployment**:
    - It constructs a new deployment manifest that includes both the new files and the existing files (referenced by hash).
    - It uploads only the new/changed files.
    - It updates the D1 database with the new state.
4.  **Result**: A new deployment on Cloudflare Pages that reflects your desired file state, achieved efficiently.
