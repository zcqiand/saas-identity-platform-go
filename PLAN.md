# 未来开发计划

- [ ] lab-go 全量业务复刻（REQ 独立批，照 laravel-lab 形状：auth 代理链分叉无 aud / 样品-目录模型）
- [ ] 书稿章节映射补齐（README「配套书籍及章节映射」）
- [ ] PKCE：契约列已在（oauth_code.code_challenge*），家族若启用校验五栈同步（本批裁定按镜像透传，见 REQ-2026-003 收账）

已交付：REQ-2026-001 健康检查 + 端口装配（v0.1.0-20261026）；REQ-2026-002 进 prod deploy 链（v0.1.1-20261026，`https://{domain}`，X13={port}）；REQ-2026-003 全量业务切片 47 op / 11 tag（v0.2.0-20261027，含 contract-test go 目标注册 + live 2-way 收敛）。
