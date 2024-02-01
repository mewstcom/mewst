# Mewst

## リンク

- [ロードマップ](https://github.com/orgs/mewsted/projects/1)
- [Discord](https://discord.gg/tNwVpJ4Jfk)
- [GitHub Issues](https://github.com/mewsted/mewst/issues)

## 開発環境のセットアップ

```
git clone git@github.com:mewsted/mewst.git
cd mewst
docker compose up
docker compose exec app bin/setup
docker compose exec app bin/dev
docker compose exec app bin/rails server
```
