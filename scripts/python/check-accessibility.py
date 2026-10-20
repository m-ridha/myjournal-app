"""Optional local browser QA; see docs/development/testing.md."""

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
    arrows = await page.locator('.mj-icon-btn:enabled, [data-header-language] [data-language-trigger]').evaluate_all("""buttons => {
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
    controlled_menus = await page.locator('[data-language-trigger]').evaluate_all("""buttons => buttons.map(button => {
        const id = button.getAttribute('aria-controls');
        const menu = document.getElementById(id);
        return {id, count: document.querySelectorAll('#' + id).length,
            role: menu && menu.getAttribute('role'),
            below: !menu || menu.hidden || menu.getBoundingClientRect().top >= button.getBoundingClientRect().bottom};
    })""")
    assert all(menu['count'] == 1 and menu['role'] == 'listbox' and menu['below']
               for menu in controlled_menus), controlled_menus
    return {"label": label, "arrow_contrast": arrows, "controlled_menus": controlled_menus, **result}


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
        calendar_requests = []
        page.on('request', lambda request: calendar_requests.append(request.url) if '/api/v1/calendar?' in request.url else None)
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
                         '/help', '/privacy', '/terms', '/sign-in', '/sign-in-failed', '/invalid-page']:
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
                if path == '/calendar':
                    await expect(page.locator('[data-calendar-next]')).to_be_disabled()
                    trigger = page.locator('[data-month-trigger]')
                    picker = page.locator('[data-month-picker]')
                    count = len(calendar_requests)
                    await trigger.click()
                    await expect(picker).to_be_visible()
                    scans.append(await audit(page, axe_script, theme + '/month-picker'))
                    await page.screenshot(path='/tmp/myjournal-month-picker-' + theme + '.png', full_page=True)
                    await page.keyboard.press('ArrowLeft')
                    await page.keyboard.press('Escape')
                    await expect(trigger).to_be_focused()
                    await trigger.click()
                    await page.locator('[data-month-year]').click()
                    scans.append(await audit(page, axe_script, theme + '/year-picker'))
                    await page.screenshot(path='/tmp/myjournal-year-picker-' + theme + '.png', full_page=True)
                    year = int((await page.locator('body').get_attribute('data-today'))[:4])
                    await page.locator('[data-month-options] button').filter(has_text=str(year - 1)).click()
                    assert len(calendar_requests) == count, 'Browsing picker fetched calendar data'
                    await page.set_viewport_size({'width': 390, 'height': 844})
                    assert await page.evaluate('document.documentElement.scrollWidth <= innerWidth')
                    scans.append(await audit(page, axe_script, theme + '/month-picker-narrow'))
                    await page.screenshot(path='/tmp/myjournal-month-picker-narrow-' + theme + '.png', full_page=True)
                    await page.set_viewport_size({'width': 320, 'height': 720})
                    assert await page.evaluate('document.documentElement.scrollWidth <= innerWidth')
                    scans.append(await audit(page, axe_script, theme + '/month-picker-320'))
                    await page.set_viewport_size({'width': 390, 'height': 844})
                    await page.locator('[data-month-options] button').nth(1).click()
                    await expect(picker).to_be_hidden()
                    await expect(trigger).to_be_focused()
                    await expect(page.locator('[data-calendar-grid]')).to_have_attribute('aria-busy', 'false')
                    assert len(calendar_requests) == count + 1, 'Month selection must fetch exactly once'
                    await expect(page.locator('[data-calendar-month]')).to_contain_text(str(year - 1))
                    await page.locator('[data-calendar-today]').click()
                    await expect(page.locator('[data-calendar-next]')).to_be_disabled()
                    await page.set_viewport_size({'width': 1280, 'height': 720})


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

            async def expired(route):
                await route.fulfill(status=401, content_type='application/json', body='{}')
            for path, endpoint in [('/entries', '**/api/v1/entries?*'),
                                   ('/calendar', '**/api/v1/calendar?*')]:
                await page.route(endpoint, expired)
                await page.goto(base + path)
                await expect(page.locator('[data-auth-expired]')).to_be_visible()
                scans.append(await audit(page, axe_script, theme + path + '/auth-expired'))
                keyboard[theme + path + '/auth-expired'] = await check_keyboard(page)
                await page.unroute(endpoint, expired)

            await page.goto(base + '/entries/2020-02-01')
            await page.wait_for_function("!document.querySelector('[data-entry-text]').readOnly")
            reconnect_text = 'Synthetic reconnect draft ' + theme
            await page.locator('[data-entry-text]').fill(reconnect_text)
            await page.route('**/api/v1/entries/2020-02-01', expired)
            await page.locator('[data-entry-save]').click()
            await expect(page.locator('[data-auth-expired]')).to_be_visible()
            await expect(page.locator('[data-entry-text]')).to_have_value(reconnect_text)
            await page.screenshot(path='/tmp/myjournal-auth-expired-' + theme + '.png', full_page=True)
            scans.append(await audit(page, axe_script, theme + '/editor-auth-expired'))
            keyboard[theme + '/editor-auth-expired'] = await check_keyboard(page)
            await page.set_viewport_size({'width': 390, 'height': 844})
            assert await page.evaluate('document.documentElement.scrollWidth <= window.innerWidth'), 'Reconnect banner overflows narrow viewport'
            await page.screenshot(path='/tmp/myjournal-auth-expired-narrow-' + theme + '.png', full_page=True)
            scans.append(await audit(page, axe_script, theme + '/editor-auth-expired-narrow'))
            await page.set_viewport_size({'width': 1280, 'height': 720})
            async with page.expect_popup() as popup_info:
                await page.locator('[data-auth-expired] a').click()
            popup = await popup_info.value
            await expect(popup.get_by_role('link', name='Continue with Local Dev', exact=True)).to_be_visible()
            await popup.get_by_role('link', name='Continue with Local Dev', exact=True).click()
            await popup.wait_for_function("!document.querySelector('[data-entry-text]').readOnly")
            await popup.close()
            await expect(page.locator('[data-entry-text]')).to_have_value(reconnect_text)
            await page.unroute('**/api/v1/entries/2020-02-01', expired)
            await page.locator('[data-entry-save]').click()
            await expect(page.locator('[data-entry-status]')).to_have_text('Saved')
            await expect(page.locator('[data-auth-expired]')).to_be_hidden()

        # A separate cookie/storage context proves these are public pages even
        # after the authenticated editor checks above have established a session.
        public_context = await browser.new_context(timezone_id='Asia/Jakarta')
        public_page = await public_context.new_page()
        public_page.on('pageerror', lambda error: errors.append(str(error)))
        for theme in ['light', 'dark']:
            await public_page.goto(base + '/')
            await public_page.evaluate('(theme) => localStorage.setItem("mj-theme", theme)', theme)
            for path in ['/', '/settings', '/help', '/privacy', '/terms', '/sign-in', '/sign-in-failed', '/invalid-page']:
                response = await public_page.goto(base + path)
                assert response.status == (404 if path == '/invalid-page' else 200), path
                assert public_page.url == base + path, public_page.url
                await expect(public_page.get_by_role('link', name='Settings', exact=True)).to_be_visible()
                if path == '/settings':
                    await expect(public_page.locator('[data-sign-out], [data-disconnect]')).to_have_count(0)
                if path == '/':
                    await expect(public_page.get_by_role('link', name='Continue with Local Dev', exact=True)).to_be_visible()
                label = 'signed-out/' + theme + path
                scans.append(await audit(public_page, axe_script, label))
                keyboard[label] = await check_keyboard(public_page)
                if path == '/settings':
                    await public_page.locator('[data-header-language] [data-language-trigger]').click()
                    scans.append(await audit(public_page, axe_script, label + '/language-open'))
                    await public_page.keyboard.press('Escape')
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
