# Leeforge Framework

`server/framework` 是 `server/frame-core` 的迁移承接模块，目标模块路径为 `github.com/leeforge/framework`。

## Scope

- 仅承载通用技术内核能力（鉴权、授权、响应封装、中间件、插件运行时等）。
- 不承载业务编排逻辑；业务实现必须位于上层业务模块并按插件优先原则接入。

## Migration Notes

- 迁移阶段允许在 monorepo 内通过 `replace github.com/leeforge/framework => ../framework` 联调。
- 完成独立仓库发布后，backend 应改为消费已发布 tag。
