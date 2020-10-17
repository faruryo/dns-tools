![Docker](https://github.com/faruryo/dns-tools/workflows/Docker/badge.svg)

## secretセットアップ

```shell
SLACK_WEBHOOK_URL="hoge"
CLOUDFLARE_API_TOKEN="fuga"

kubectl create secret generic dns-tools \
    --from-literal=SLACK_WEBHOOK_URL=$SLACK_WEBHOOK_URL \
    --from-literal=CLOUDFLARE_API_TOKEN=$CLOUDFLARE_API_TOKEN
```

## 参考

- [Build a GraphQL API in Golang with MySQL and GORM using Gqlgen | SoberKoder](https://www.soberkoder.com/go-graphql-api-mysql-gorm/)
- [gqlgen + Gorm でUint型の場合エラーになる - Qiita](https://qiita.com/3104k/items/caf17633d4926aee8a84)
- [Golang テスト sqlmock | 実務のGo](https://www.go-lang-programming.com/doc/test/sqlmock)
