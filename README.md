# docblog

This tool parses all Google Docs from a specific Google Drive directory into
HTML files that can be later published using a static site generator tool such
as [Jekyll] or [Hugo]. For a usage example see my blog, [jupblb.github.io].

This tool also creates a Google Sheet inside the Google Drive directory through
which it’s possible to modify document metadata, such as publication time or
description.

## Usage

``` sh
go run cmd/docblog/main.go \
  --assets-output website/assets \
  --posts-output website/posts \
  --credentials $CREDENTIALS_FILE_PATH \
  $DRIVE_DIRECTORY_ID
```

All the optional flags can be discovered by using the `--help` flag.

## Google Cloud auth

The credentials file must be obtained in one of the following ways:

1.  Use `gcloud` CLI to generate the file

    ``` sh
    gcloud auth application-default login \
      --client-id-file=$CREDENTIALS_FILE_PATH \
      --scopes="https://www.googleapis.com/auth/docs" \
      --scopes="https://www.googleapis.com/auth/drive" \
      --scopes="https://www.googleapis.com/auth/drive.file" \
      --scopes="https://www.googleapis.com/auth/drive.metadata" \
      --scopes="https://www.googleapis.com/auth/spreadsheets"
    ```

2.  Create a service account and download the credentials file. Remember to
    share the Google Drive directory with that service account.

  [Jekyll]: https://jekyllrb.com
  [Hugo]: https://gohugo.io
  [jupblb.github.io]: https://github.com/jupblb/jupblb.github.io
