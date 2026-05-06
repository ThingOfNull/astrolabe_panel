import { expect, test } from '@playwright/test';

// 回归测试：确认 widget 在设置页和主页上处于一致的"视觉位置"。
//
// 核心防御点：
//   - SettingsPage 必须在 RPC 重连时重新拉取 widgets（避免页面刷新后空看板）；
//   - 设置页画布的视觉描边层不能影响内部 Canvas 布局；
//   - widget 在两页 canvas 内部的相对横向位置（canvas-relative center）应当相同；
//   - 主页画布需自适应缩放并水平居中，以确保 viewport-center ≈ canvas-center，
//     否则窄屏下"画布右侧组件"会因画布溢出而看起来"穿越视觉中心线"。

test.describe('widget position parity', () => {
  test('widget at right-of-center keeps the same canvas-relative position on both pages', async ({
    page,
  }) => {
    await page.goto('/');
    await page.waitForFunction(
      () => Boolean((window as unknown as { __ASTROLABE_TEST_RPC?: unknown }).__ASTROLABE_TEST_RPC),
      { timeout: 10_000 },
    );

    // 在 canvas 右半部分插入一个 widget。canvas 设计宽度 200 网格单位，
    // x=140, w=20 → 中心点 = 150 个单位 → relCenterFrac = 0.75。
    const created = await page.evaluate(async () => {
      const win = window as unknown as {
        __ASTROLABE_TEST_RPC: (m: string, p: unknown) => Promise<unknown>;
      };
      const result = (await win.__ASTROLABE_TEST_RPC('widget.create', {
        type: 'link',
        x: 140,
        y: 50,
        w: 20,
        h: 8,
        icon_type: 'ICONIFY',
        icon_value: 'mdi:web',
        config: {
          title: 'POS-CHECK',
          url: 'https://example.com',
          open_in_new_tab: true,
          probe: { enabled: false, type: 'http', interval_sec: 30, timeout_sec: 4 },
        },
      })) as { id: number };
      return result.id;
    });

    async function measure(): Promise<{
      canvasFrac: number;
      widgetCenterX: number;
      canvasCenterX: number;
      viewportCenterX: number;
    }> {
      await page.waitForSelector('[data-widget-id]', { state: 'attached', timeout: 10_000 });
      await page.waitForTimeout(200);
      return page.evaluate(() => {
        const widget = document.querySelector('[data-widget-id]') as HTMLElement;
        const canvas = document.querySelector('.canvas-root') as HTMLElement;
        const w = widget.getBoundingClientRect();
        const c = canvas.getBoundingClientRect();
        return {
          canvasFrac: (w.left + w.width / 2 - c.left) / c.width,
          widgetCenterX: w.left + w.width / 2,
          canvasCenterX: c.left + c.width / 2,
          viewportCenterX: window.innerWidth / 2,
        };
      });
    }

    await page.goto('/');
    const home = await measure();

    await page.goto('/settings');
    const settings = await measure();

    await page.evaluate(async (id: number) => {
      const win = window as unknown as {
        __ASTROLABE_TEST_RPC: (m: string, p: unknown) => Promise<unknown>;
      };
      await win.__ASTROLABE_TEST_RPC('widget.delete', { id });
    }, created);

    // 1）widget 中心点在 canvas 内部位置一致（皆为 75%）
    expect(home.canvasFrac).toBeCloseTo(0.75, 2);
    expect(settings.canvasFrac).toBeCloseTo(0.75, 2);
    expect(Math.abs(home.canvasFrac - settings.canvasFrac)).toBeLessThan(0.01);

    // 2）主页画布水平居中：canvas-center 与 viewport-center 偏差小（< 视口的 5%）
    expect(Math.abs(home.canvasCenterX - home.viewportCenterX)).toBeLessThan(
      0.05 * 1280,
    );

    // 3）画布右侧 widget 在主页上必须依然位于 viewport-center 右侧，
    //    即不会因主页画布溢出而被误移到中心线左边。
    expect(home.widgetCenterX).toBeGreaterThan(home.viewportCenterX);
  });
});
