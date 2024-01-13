# mewst-api

[Mewst](https://www.mewst.com)のバックエンドです。

## 開発環境のセットアップ

```
git clone git@github.com:mewsted/mewst-api.git
cd mewst-api
docker compose up
docker compose exec app bin/setup
docker compose exec app bin/rails server
```

Sorbetを実行するにはローカル環境で `srb` コマンドを実行する必要があります。

```
mise trust && mise install
bundle install
bin/srb tc
```
