import { expect, test } from '@playwright/test';

// 关键路径 E2E：
//
//   1. 加载首页 → 显示空看板提示。
//   2. 点击 "去设置" → URL 跳到 /settings。
//   3. 通过 RPC 注入一个 link widget（避免与 interactjs 拖拽细节耦合）。
//   4. 回到首页 → 应能看到 link widget 标题。
//
// 后端 / 前端由 Makefile + tests/e2e/run-server.sh 启动好（监听 18080）；
// 这里仅访问 Web。

test.describe('astrolabe core flow', () => {
  test('empty board → settings → create link → home shows widget', async ({ page, request }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: /astrolabe/i })).toBeVisible();
    // 空看板提示
    await expect(page.getByText(/去设置页|拖|前往设置|empty/i).first()).toBeVisible({
      timeout: 10_000,
    });

    // 点击 "去设置" / "Open Settings"
    const settingsBtn = page.getByRole('button', { name: /设置|Settings/i }).first();
    await settingsBtn.click();
    await expect(page).toHaveURL(/\/settings/);
    // palette 三个 tab 之一
    await expect(page.getByText('组件库').first()).toBeVisible();

    // 走后端 healthz 验证服务可用，再用 RPC（WebSocket 无法直接 fetch；
    // 这里通过浏览器执行 evaluate 调用前端 store 创建 link，复用现有连接）。
    const created = await page.evaluate(async () => {
      const win = window as unknown as {
        __ASTROLABE_TEST_RPC?: (m: string, p: unknown) => Promise<unknown>;
      };
      const call = win.__ASTROLABE_TEST_RPC;
      if (!call) throw new Error('test RPC bridge missing');
      const result = (await call('widget.create', {
        type: 'link',
        x: 0,
        y: 0,
        w: 12,
        h: 8,
        icon_type: 'ICONIFY',
        icon_value: 'mdi:web',
        config: {
          title: 'E2E Demo',
          url: 'https://example.com',
          open_in_new_tab: true,
          probe: { enabled: false, type: 'http', interval_sec: 30, timeout_sec: 4 },
        },
      })) as { id: number };
      return result.id;
    });
    expect(typeof created).toBe('number');

    // 回首页验证
    await page.goto('/');
    await expect(page.getByText('E2E Demo').first()).toBeVisible({ timeout: 10_000 });

    // 清理：删掉刚才创建的 widget，避免下次跑测试时被空看板断言挡住
    await page.evaluate(async (id: number) => {
      const win = window as unknown as {
        __ASTROLABE_TEST_RPC?: (m: string, p: unknown) => Promise<unknown>;
      };
      await win.__ASTROLABE_TEST_RPC?.('widget.delete', { id });
    }, created);
  });

  test('healthz is reachable', async ({ request }) => {
    const resp = await request.get('/healthz');
    expect(resp.ok()).toBeTruthy();
    const body = (await resp.json()) as { ok?: boolean };
    expect(body.ok).toBe(true);
  });
});
