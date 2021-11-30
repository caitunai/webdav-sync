Webdav sync
-----
> Auto upload or delete file from webdav server as what changed in the local disk.

## Install
```shell
> go install github.com/caitunai/webdav-sync@v1.1.1
```

## Configuration
> You should have a configuration file `webdav.json` in the directory where you run `webdav-sync`.

**webdav.json**
```json
{
  "local_path": "/Users/user/Documents/path/to/upload",
  "server": "http://webdav.example.com:8099",
  "server_path": "path/of/the/webdav/server",
  "username": "webdavuser",
  "password": "webdavpassword",
  "ignores": [
    ".git",
    ".idea"
  ]
}
```

## Run
> Run webdav-sync command in the terminal at the path of webdav.json
```shell
> webdav-sync
```