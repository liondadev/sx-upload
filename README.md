# ShareX Uploading Host

This was made becuase I need a place to store my photos after upload.systems shutdown.

## Features

- Full ShareX Support
- Multi User Support
- Dashboard for each user to see their uploads
- Allow users to export their entire upload history into a zip file
- SQLite for data & file storage, no external database required

## Installation (Docker)

> [!NOTICE]
> The application listens on port 8080, you will need to bind a port to this port when running the container to expose it.

Docker is the only "officially" supported way to run this project. You can run it without Docker, but you're on your own.

1. Setup a dockerfile

```
    TODO!
```

## Configuration File

The app runs off a configuration file. The path of the configuration file is defined in the `SX_UPLOAD_CONFIG_PATH` environment variable.

The config should look something like this:

```json
{
  "users": {
    "user1": {
      "max_upload_size": 450000 // max file upload size, in bytes
    }
  },
  "keys": {
    "super_secret_key": "user1"
  },
  "admin_token": "even_more_secret_key"
}
```

### Users

Users are identified with an ID, the key in the users object.

### Keys

Keys are used to authenticate users. The key is the key in the keys object, and the value is the user ID.

### Admin Token

The admin token is currently unused, but may be used in the future for viewing ALL uploads on the dashboard.

## Environment Variables

- **SX_UPLOAD_CONFIG_PATH**: Path to the configuration file
- **SX_BASE_URL**: The base URL to the instance. (e.g. `https://upload.example.com`) - DOES NOT CONTAIN A TRAILING SLASH
- **SX_UPLOAD_SQLITE_LOCATION**: The location of the SQLite database file. If not set, will store everything in memory.
- **SX_STATIC_DIR**: You don't need to set this, the container does it for you. The directory where the static files for the dashboard are stored.
