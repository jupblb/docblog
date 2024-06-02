# Google credentials

``` bash
$ gcloud auth login
$ gcloud config set project $PROJECT_ID
# Setup consent screen via cloud console (remember to enable relevant APIs)
# Setup OAuth Client ID and download credentials to $CREDENTIALS_FILE_PATH
gcloud auth application-default login \
  --client-id-file=$CREDENTIALS_FILE_PATH \
  --scopes="https://www.googleapis.com/auth/docs" \
  --scopes="https://www.googleapis.com/auth/drive" \
  --scopes="https://www.googleapis.com/auth/drive.file" \
  --scopes="https://www.googleapis.com/auth/drive.metadata" \
  --scopes="https://www.googleapis.com/auth/spreadsheets"
```
