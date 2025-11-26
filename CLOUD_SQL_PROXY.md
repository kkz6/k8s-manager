# Cloud SQL Proxy Feature

## Overview

The K8s Manager now includes a Cloud SQL Proxy feature that allows you to connect to private Cloud SQL databases that are only accessible within your VPC.

## Prerequisites

1. **Install Cloud SQL Proxy**:
   ```bash
   gcloud components install cloud-sql-proxy
   ```

2. **GCP Authentication**:
   - Ensure you're authenticated with gcloud:
     ```bash
     gcloud auth login
     gcloud auth application-default login
     ```

3. **IAM Permissions**:
   - Your GCP account needs the `Cloud SQL Client` role:
     ```bash
     gcloud projects add-iam-policy-binding PROJECT_ID \
       --member="user:YOUR_EMAIL" \
       --role="roles/cloudsql.client"
     ```

4. **Network Access for Private IP Instances**:
   - If your Cloud SQL instance only has a **private IP** (VPC-only), you need one of:
     - Enable **Public IP** on the instance (recommended for development)
     - Set up **Cloud VPN** or **Cloud Interconnect** to access the VPC
     - Run from a **GCE VM** or **GKE cluster** in the same VPC
   - The cloud-sql-proxy cannot connect to private IPs from your local machine without VPC access

## How to Use

### 1. Access the Cloud SQL Proxy Manager

From the main menu:
- Press **'x'** or select **"Cloud SQL Proxy"**

### 2. Start a Proxy Connection

1. The application will list all Cloud SQL instances in your project
2. Select an instance from the list
3. Press **'s'** or **Enter** to start the proxy
4. The proxy will start on an available local port (starting from 3306)
5. You'll see the status change to "🟢 Running on localhost:PORT"

### 3. Connect to Your Database

Once the proxy is running, you can connect to your database using:

**MySQL:**
```bash
mysql -h 127.0.0.1 -P 3306 -u your_user -p
```

**PostgreSQL:**
```bash
psql -h 127.0.0.1 -p 3306 -U your_user -d your_database
```

**From Your Application:**
Update your database connection settings:
```
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USERNAME=your_user
DB_PASSWORD=your_password
DB_DATABASE=your_database
```

### 4. Open in TablePlus (Optional)

If you have TablePlus installed, you can automatically open a connection:
1. Select a running instance
2. Press **'o'** to open in TablePlus
3. TablePlus will open with the connection pre-configured
4. You can then enter your database credentials in TablePlus

### 5. Stop a Proxy Connection

- Select the running instance
- Press **'x'** to stop the proxy
- Or press **'a'** to stop all running proxies

## Key Bindings

| Key | Action |
|-----|--------|
| **s** or **Enter** | Start proxy for selected instance |
| **x** | Stop proxy for selected instance |
| **o** | Open in TablePlus (if running) |
| **a** | Stop all running proxies |
| **r** | Refresh instance list |
| **q/b/esc** | Go back to main menu |
| **ctrl+c** | Quit (stops all proxies) |

## Features

- ✅ List all Cloud SQL instances in your project
- ✅ Start/stop proxy connections
- ✅ Automatic port assignment (avoids conflicts)
- ✅ Multiple simultaneous connections
- ✅ Real-time status monitoring
- ✅ Automatic cleanup on exit
- ✅ One-click TablePlus integration

## Port Management

The application automatically assigns ports starting from 3306. If a port is already in use, it will increment to the next available port (3307, 3308, etc.).

## Troubleshooting

### "cloud-sql-proxy not found"

Install the Cloud SQL Proxy:
```bash
gcloud components install cloud-sql-proxy
```

If you still get the error after installing:
1. Restart your terminal to refresh PATH
2. Or add the gcloud SDK to your PATH:
   ```bash
   # For bash/zsh
   export PATH="$PATH:/opt/homebrew/share/google-cloud-sdk/bin"

   # Or find your gcloud SDK path and add it
   gcloud info --format="value(installation.sdk_root)"
   export PATH="$PATH:$(gcloud info --format='value(installation.sdk_root)')/bin"
   ```
3. The application will automatically search common installation paths:
   - `/opt/homebrew/share/google-cloud-sdk/bin/` (Homebrew macOS)
   - `/usr/local/google-cloud-sdk/bin/` (Standard macOS)
   - `/usr/bin/` and `/usr/local/bin/` (Linux)

### "Permission denied" errors

Ensure you have the Cloud SQL Client role:
```bash
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="user:YOUR_EMAIL" \
  --role="roles/cloudsql.client"
```

### Connection timeout

1. Verify your Cloud SQL instance is running
2. Check that your GCP credentials are valid:
   ```bash
   gcloud auth application-default login
   ```

### Port already in use

The application will automatically find the next available port. If you need a specific port, stop other services using that port first.

## Security Notes

- Connections are proxied through your local machine
- The proxy uses your GCP credentials for authentication
- All traffic is encrypted using Cloud SQL's SSL/TLS
- The proxy only runs while the application is open
- All proxies are automatically stopped when you exit

## TablePlus Integration

If you have TablePlus installed, you can use the built-in integration:

1. Start a proxy connection (press 's' or Enter)
2. Once running, press **'o'** to open in TablePlus
3. TablePlus will launch with the connection pre-configured:
   - Host: 127.0.0.1
   - Port: (auto-assigned port, e.g., 3306)
   - Type: MySQL
4. Enter your database credentials in TablePlus
5. Save the connection for future use

**Note**: This feature uses the `tableplus://` URL scheme. If you don't have TablePlus installed, you can:
- Download it from: https://tableplus.com/
- Or continue using the command-line method below

## Example Workflow

### With TablePlus:
1. Start k8s-manager: `./k8s-manager`
2. Press **'x'** to access Cloud SQL Proxy
3. Select your database instance
4. Press **'s'** to start the proxy
5. Press **'o'** to open in TablePlus
6. Enter your credentials in TablePlus
7. When done, return to k8s-manager and press **'x'** to stop

### With Command Line:
1. Start k8s-manager: `./k8s-manager`
2. Press **'x'** to access Cloud SQL Proxy
3. Select your database instance
4. Press **'s'** to start the proxy
5. Note the local port (e.g., 3306)
6. Open a new terminal and connect:
   ```bash
   mysql -h 127.0.0.1 -P 3306 -u myuser -p
   ```
7. When done, return to k8s-manager and press **'x'** to stop

## Configuration

The Cloud SQL Proxy uses your project configuration from `k8s-manager.yaml`:
- `gcp.project_id`: Used to list and connect to instances
- `gcp.region`: Used for regional instance queries

You can update these settings via the Configuration menu (press 'g').
