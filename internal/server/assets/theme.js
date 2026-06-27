// MiniSnap 前端：全站主题持久化 + 阅读页字号调节。
// 所有页面通过 <script src="/-/static/theme.js" defer> 引入。
// 防 FOUC 的初始 data-theme 由各模板 head 内联脚本提前设置。
(function () {
	const THEME_KEY = 'minisnap.theme';
	const FONT_KEY = 'minisnap.font';
	const FONT_STEPS = ['sm', 'md', 'lg', 'xl'];

	const root = document.documentElement;

	const Minisnap = {
		// 应用主题并更新所有切换按钮的图标
		applyTheme(theme) {
			root.setAttribute('data-theme', theme);
			document.querySelectorAll('[data-theme-toggle]').forEach((btn) => {
				const icon = btn.querySelector('.icon');
				if (icon) icon.textContent = theme === 'dark' ? '🌚' : '🌞';
			});
		},
		// 在亮/暗之间切换并持久化
		toggleTheme() {
			const next = root.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
			this.applyTheme(next);
			try { window.localStorage.setItem(THEME_KEY, next); } catch (e) {}
		},
		// 初始化主题切换按钮（任意页面）
		initThemeToggles() {
			document.querySelectorAll('[data-theme-toggle]').forEach((btn) => {
				btn.addEventListener('click', () => this.toggleTheme());
			});
		},
		// === 字号：仅阅读页调用 ===
		applyFont(step) {
			root.setAttribute('data-font', step);
			const idx = FONT_STEPS.indexOf(step);
			const down = document.getElementById('font-down');
			const up = document.getElementById('font-up');
			if (down) down.disabled = idx <= 0;
			if (up) up.disabled = idx >= FONT_STEPS.length - 1;
		},
		changeFont(delta) {
			let idx = FONT_STEPS.indexOf(root.getAttribute('data-font'));
			if (idx === -1) idx = FONT_STEPS.indexOf('md');
			idx = Math.min(FONT_STEPS.length - 1, Math.max(0, idx + delta));
			const step = FONT_STEPS[idx];
			this.applyFont(step);
			try { window.localStorage.setItem(FONT_KEY, step); } catch (e) {}
		},
		// 阅读页初始化字号（按钮 + 持久化）
		initFontControls() {
			const down = document.getElementById('font-down');
			const up = document.getElementById('font-up');
			if (!down || !up) return;
			let saved = null;
			try { saved = window.localStorage.getItem(FONT_KEY); } catch (e) {}
			this.applyFont(FONT_STEPS.indexOf(saved) !== -1 ? saved : 'md');
			down.addEventListener('click', () => this.changeFont(-1));
			up.addEventListener('click', () => this.changeFont(1));
		},
	};

	window.Minisnap = Minisnap;

	document.addEventListener('DOMContentLoaded', function () {
		Minisnap.initThemeToggles();
		Minisnap.initFontControls();
	});
})();
