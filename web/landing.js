/**
 * ShortURL: Landing Page
 * Theme toggle and the shorten form. The link card starts with a sample
 * link and is replaced by the real one after a successful shorten.
 */

/* ── Theme ── */
function getPreferredTheme() {
  const stored = localStorage.getItem('su_theme');
  if (stored === 'light' || stored === 'dark') return stored;
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function setTheme(theme) {
  document.documentElement.classList.toggle('dark', theme === 'dark');
  localStorage.setItem('su_theme', theme);
}

document.addEventListener('DOMContentLoaded', () => {
  setTheme(getPreferredTheme());

  document.getElementById('theme-toggle-landing')?.addEventListener('click', () => {
    const isDark = document.documentElement.classList.contains('dark');
    setTheme(isDark ? 'light' : 'dark');
  });

  /* ── Shorten form ── */
  const form = document.getElementById('shorten-form');
  const group = document.getElementById('shorten-group');
  const input = document.getElementById('demo-input');
  const button = document.getElementById('demo-btn');
  const hint = document.getElementById('demo-hint');
  const error = document.getElementById('demo-error');

  const card = document.getElementById('link-card');
  const label = document.getElementById('link-label');
  const shortLink = document.getElementById('link-short');
  const dest = document.getElementById('link-dest');
  const stats = document.getElementById('link-stats');
  const fresh = document.getElementById('link-fresh');
  const copyBtn = document.getElementById('copy-btn');

  function showError(message) {
    group.classList.add('is-error');
    error.textContent = message;
    error.hidden = false;
    hint.hidden = true;
  }

  function clearError() {
    group.classList.remove('is-error');
    error.hidden = true;
    hint.hidden = false;
  }

  function showLink(shortUrl, originalUrl) {
    card.classList.add('is-real');
    label.textContent = 'Your link';
    shortLink.textContent = shortUrl.replace(/^https?:\/\//, '');
    shortLink.href = shortUrl;
    shortLink.removeAttribute('tabindex');
    dest.textContent = '→ ' + originalUrl.replace(/^https?:\/\//, '');
    stats.hidden = true;
    fresh.hidden = false;
  }

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    const url = input.value.trim();

    if (!url) {
      showError('Paste a URL first.');
      input.focus();
      return;
    }

    clearError();
    button.disabled = true;
    button.textContent = 'Shortening…';

    try {
      const res = await fetch('/api/v1/shorten', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ original_url: url }),
        credentials: 'same-origin',
      });

      if (res.status === 429) {
        showError('You’ve used your 3 free links. Create an account to keep shortening.');
      } else if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        showError(data.message || 'Couldn’t shorten that link. Check the URL and try again.');
      } else {
        const data = await res.json();
        const shortUrl = data.data && data.data.short_url;
        if (shortUrl) showLink(shortUrl, url);
        else showError('Something went wrong. Please try again.');
      }
    } catch (err) {
      showError('Network error. Check your connection and try again.');
    }

    button.disabled = false;
    button.textContent = 'Shorten';
  });

  input.addEventListener('input', () => {
    if (!error.hidden) clearError();
  });

  copyBtn.addEventListener('click', () => {
    const text = shortLink.href && card.classList.contains('is-real') ? shortLink.href : 'https://' + shortLink.textContent;
    navigator.clipboard.writeText(text).then(() => {
      copyBtn.textContent = 'Copied';
      copyBtn.classList.add('is-copied');
      setTimeout(() => {
        copyBtn.textContent = 'Copy';
        copyBtn.classList.remove('is-copied');
      }, 2000);
    });
  });
});
