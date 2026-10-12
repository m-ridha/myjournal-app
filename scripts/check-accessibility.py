"""Optional local browser QA; see docs/development/ui-acceptance-checks.md."""

import argparse
import asyncio
import base64
import json
import os
from pathlib import Path
import signal
import socket
import subprocess
import tempfile

from playwright.async_api import async_playwright, expect


async def audit(page, axe_script, label):
    await page.evaluate(axe_script)
    result = await page.evaluate("""async () => {
        const r = await axe.run(document, {runOnly: {type: 'tag',
            values: ['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa']}});
        return {violations: r.violations, incomplete: r.incomplete,
            passes: r.passes.map(p => p.id)};
    }""")
    arrows = await page.locator('.mj-icon-btn:enabled').evaluate_all("""buttons => {
        const rgb = value => (value.match(/[0-9.]+/g) || []).map(Number);
        const luminance = color => rgb(color).slice(0, 3).map(v => {
            v /= 255; return v <= .04045 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4;
        }).reduce((sum, v, i) => sum + v * [.2126, .7152, .0722][i], 0);
        return buttons.map(button => {
            let parent = button;
            while (parent && rgb(getComputedStyle(parent).backgroundColor)[3] === 0)
                parent = parent.parentElement;
            const foreground = getComputedStyle(button).color;
            const background = getComputedStyle(parent).backgroundColor;
            const a = luminance(foreground), b = luminance(background);
            return {label: button.getAttribute('aria-label'), foreground, background,
                ratio: (Math.max(a, b) + .05) / (Math.min(a, b) + .05)};
        });
    }""")
    assert all(arrow['ratio'] >= 4.5 for arrow in arrows), arrows
    return {"label": label, "arrow_contrast": arrows, **result}


async def check_keyboard(page):
    controls = page.locator('a[href], button:enabled, textarea:not([readonly])')
    count = await controls.count()
    expected = set(await controls.evaluate_all("""elements => elements
        .filter(e => e.getClientRects().length > 0 && getComputedStyle(e).visibility !== 'hidden')
        .map(e => e.getAttribute('aria-label') || e.textContent.trim())"""))
    visited = set()
    for _ in range(count + 2):
        await page.keyboard.press('Tab')
        focus = await page.evaluate("""() => {
            const e = document.activeElement;
            if (e === document.body) return null;
            const s = getComputedStyle(e);
            const editor = e.matches('[data-entry-text]');
            return {name: e.getAttribute('aria-label') || e.textContent.trim(),
                keyboard: e.matches(':focus-visible'),
                indicator: (s.outlineStyle !== 'none' && parseFloat(s.outlineWidth) > 0)
                    || (editor && getComputedStyle(e.parentElement).borderColor === s.color)};
        }""")
        if focus:
            assert focus['keyboard'] and focus['indicator'], focus
            visited.add(focus['name'])
    assert expected and expected <= visited, {'unreached': sorted(expected - visited)}
    return sorted(visited)


async def run_browser(base, axe_script):
    scans, keyboard, errors = [], {}, []
    async with async_playwright() as p:
        browser = await p.chromium.launch(channel='chrome', headless=True)
        context = await browser.new_context(timezone_id='Asia/Jakarta')
        page = await context.new_page()
        page.on('pageerror', lambda error: errors.append(str(error)))
        for _ in range(100):
            try:
                if (await context.request.get(base + '/healthz')).status == 200:
                    break
            except Exception:
                pass
            await asyncio.sleep(.1)
        else:
            raise RuntimeError('Owned test server did not become healthy')

        await page.goto(base + '/api/v1/auth/local/login')
        await page.wait_for_function("!document.querySelector('[data-entry-text]').readOnly")
        for theme in ['light', 'dark']:
            await page.evaluate('(theme) => localStorage.setItem("mj-theme", theme)', theme)
            for path in ['/entries/today', '/entries', '/calendar', '/settings',
                         '/help', '/privacy', '/terms', '/invalid-page']:
                await page.goto(base + path)
                if path == '/entries/today':
                    await page.wait_for_function("!document.querySelector('[data-entry-text]').readOnly")
                    await page.locator('[data-entry-text]').fill('Synthetic accessibility check')
                    await page.locator('[data-entry-save]').click()
                    await expect(page.locator('[data-entry-status]')).to_have_text('Saved')
                if path == '/calendar':
                    await expect(page.locator('[data-calendar-grid]')).to_have_attribute('aria-busy', 'false')
                if path == '/entries':
                    await page.wait_for_function("document.querySelector('[data-entry-list]').getAttribute('aria-busy') === 'false'")
                scans.append(await audit(page, axe_script, theme + path))
                keyboard[theme + path] = await check_keyboard(page)

            # A deterministic past date avoids changing the fixture at midnight.
            await page.goto(base + '/entries/2020-01-01')
            await page.wait_for_function("!document.querySelector('[data-entry-text]').readOnly")
            async def fail_save(route):
                await route.fulfill(status=503, content_type='application/json', body='{}')
            await page.route('**/api/v1/entries/2020-01-01', fail_save)
            await page.locator('[data-entry-text]').fill('Synthetic retained draft')
            await page.locator('[data-entry-save]').click()
            await expect(page.locator('[data-entry-status]')).to_contain_text('Save failed')
            scans.append(await audit(page, axe_script, theme + '/editor-failed-save'))
            await page.unroute('**/api/v1/entries/2020-01-01', fail_save)
            api = base + '/api/v1/entries/2020-01-01'
            current = await (await context.request.get(api)).json()
            response = await context.request.put(api, data={
                'text': 'Synthetic external update ' + theme, 'baseVersion': current['version']})
            assert response.status == 200
            await page.locator('[data-entry-save]').click()
            await expect(page.locator('[data-entry-conflict]')).to_be_visible()
            scans.append(await audit(page, axe_script, theme + '/editor-conflict'))
            keyboard[theme + '/editor-conflict'] = await check_keyboard(page)

        # A separate cookie/storage context proves these are public pages even
        # after the authenticated editor checks above have established a session.
        public_context = await browser.new_context(timezone_id='Asia/Jakarta')
        public_page = await public_context.new_page()
        public_page.on('pageerror', lambda error: errors.append(str(error)))
        for theme in ['light', 'dark']:
            await public_page.goto(base + '/')
            await public_page.evaluate('(theme) => localStorage.setItem("mj-theme", theme)', theme)
            for path in ['/', '/help', '/privacy', '/terms', '/invalid-page']:
                response = await public_page.goto(base + path)
                assert response.status == (404 if path == '/invalid-page' else 200), path
                assert public_page.url == base + path, public_page.url
                await expect(public_page.get_by_role('link', name='Settings', exact=True)).to_have_count(0)
                if path == '/':
                    await expect(public_page.get_by_role('link', name='Continue with Local Dev', exact=True)).to_be_visible()
                label = 'signed-out/' + theme + path
                scans.append(await audit(public_page, axe_script, label))
                keyboard[label] = await check_keyboard(public_page)
        await public_context.close()

        report = {'browser': browser.version, 'axe': await page.evaluate('axe.version'),
                  'scans': scans, 'keyboard': keyboard, 'page_errors': errors}
        await browser.close()
        return report


async def main(args):
    axe_script = args.axe_script.read_text()
    with tempfile.TemporaryDirectory(prefix='mj-a11y-') as data:
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            port = sock.getsockname()[1]
        base = f'http://localhost:{port}'
        env = os.environ.copy()
        env.update(PORT=str(port), APP_BASE_URL=base, PROVIDERS='local',
                   LOCAL_DATA_DIR=data, COOKIE_SECURE='false',
                   SESSION_KEY=base64.b64encode(os.urandom(32)).decode())
        with tempfile.TemporaryFile(mode='w+') as log:
            process = subprocess.Popen([str(args.app_binary.resolve())], env=env,
                                       stdout=log, stderr=log)
            try:
                report = await run_browser(base, axe_script)
                args.report.write_text(json.dumps(report, indent=2))
                for scan in report['scans']:
                    print(scan['label'], 'violations:', [v['id'] for v in scan['violations']],
                          'review:', [v['id'] for v in scan['incomplete']])
                assert not report['page_errors'], report['page_errors']
                assert not any(s['violations'] for s in report['scans']), 'Violations; see report'
                assert not any(i['id'] == 'aria-prohibited-attr'
                               for s in report['scans'] for i in s['incomplete']), 'Naming requires review'
                print(f"PASS {len(report['scans'])} scans and keyboard focus checks")
            finally:
                process.send_signal(signal.SIGTERM)
                try:
                    process.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--app-binary', required=True, type=Path)
    parser.add_argument('--axe-script', required=True, type=Path)
    parser.add_argument('--report', required=True, type=Path)
    asyncio.run(main(parser.parse_args()))
