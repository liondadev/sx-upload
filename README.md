# ShareX Uploading Host

This was made becuase I need a place to store my photos after upload.systems shutdown.

## Features

- Full ShareX Support
- Multi User Support
- Dashboard for each user to see their uploads
- Allow users to export their entire upload history into a zip file
- SQLite for data & file storage, no external database required

## Installation (Docker)

> [!NOTE]
> The application listens on port 8080, you will need to bind a port to this port when running the container to expose it.

Docker is the only "officially" supported way to run this project. You can run it without Docker, but you're on your own.

1. Create a docker compose file with the service and environment variables

```yaml
version: "3"

services:
  sxupload:
    image: ghcr.io/liondadev/sx-upload:release
    volumes:
      - "./image.config.json:/config.json:ro"
      - "sxupload_data_dir:/data"
    environment:
      SX_UPLOAD_CONFIG_PATH: /config.json
      SX_BASE_URL: https://img.example.com
      SX_UPLOAD_SQLITE_LOCATION: /data/files.db

volumes:
  sxupload_data_dir:
```

2. Create a configuration file

```yaml
# ./image.config.json

{
  "users": { "user1": { "max_upload_size": 450000 } },
  "keys": { "long-boring-key": "user1" },
  "admin_token": "longer-boring-key",
}
```

3. Deploy It

4. Go to the website in your browser, and put the key of the user you want to login as in the text field in the top right corner, and press the "Save + Refresh" button.

5. Use ShareX (see the ShareX Config section)

### ShareX Configuration

To use this we need to use a ShareX custom destination. To set this up, copy the following text to your clipboard:

```json
{
  "Version": "16.1.0",
  "Name": "img.example.com - production",
  "DestinationType": "ImageUploader",
  "RequestMethod": "POST",
  "RequestURL": "https://img.example.com/upload",
  "Headers": {
    "X-SX-API-KEY": "super_secret_key"
  },
  "Body": "MultipartFormData",
  "FileFormName": "file",
  "URL": "{json:data.link}",
  "ThumbnailURL": "{json:data.link}",
  "DeletionURL": "{json:data.delete}",
  "ErrorMessage": "{json:message}"
}
```

Go into ShareX, and on the sidebar go to the Custom uploader settings menu by clicking _Destinations_ -> _Custom uploader settings_.

Then, press the import button on the left sidebar and click "Import From Clipboard"

Then, Change the following fields accordingly:

- **Name**: Make this fit your service better
- **The API Key Header**: Set this to the key defined in the config for your user, in our example, it would be `long-boring-key`
- **The request URL**: Change the URL to the URL of your deployment

Finally, you can change the options in the bottom left to all say the newly created uploader. Don't change the link shortening ones, this service doesn't support that _yet_.

Finally Finally, you need to go back to the main ShareX menu and change all the destinations in the destinations menu to "Custom [x] uploader"

Finally Finally Finally, profit!!!

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
    "super_secret_key": "user1" // try not to include '{' or '}' in the key, ShareX doesn't like them
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
