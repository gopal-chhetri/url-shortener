/**
 * ShortURL: Landing Page Interactive Elements
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

  /* ── Interactive Terminal Demo ── */
  const demoInput = document.getElementById('demo-input');
  const demoBtn = document.getElementById('demo-btn');
  const demoOutput = document.getElementById('demo-output');
  const resultLink = document.getElementById('result-link');
  const copyBtn = document.getElementById('copy-btn');
  const processingText = document.querySelector('.processing');
  const demoHint = document.getElementById('demo-hint');

  demoBtn.addEventListener('click', async () => {
    const url = demoInput.value.trim();

    if (!url) {
      demoInput.style.border = '1px solid #ff5f56';
      setTimeout(() => demoInput.style.border = 'none', 1000);
      return;
    }

    demoBtn.disabled = true;
    demoBtn.textContent = '...';
    demoOutput.classList.remove('hidden');
    resultLink.parentElement.style.opacity = '0';
    processingText.innerHTML = '<span class="trace-dot"></span> Shortening...';

    try {
      const res = await fetch('/api/v1/shorten', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ original_url: url }),
        credentials: 'same-origin',
      });

      if (res.status === 429) {
        setError('You\u2019ve used your free links. Create an account to keep shortening.');
        demoBtn.disabled = false;
        demoBtn.textContent = 'Shorten';
        return;
      }

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setError(data.message || 'Couldn\u2019t shorten that link. Please try again.');
        demoBtn.disabled = false;
        demoBtn.textContent = 'Shorten';
        return;
      }

      const data = await res.json();
      const shortUrl = data.data && data.data.short_url;
      if (!shortUrl) {
        setError('Something went wrong. Please try again.');
        demoBtn.disabled = false;
        demoBtn.textContent = 'Shorten';
        return;
      }

      processingText.innerHTML = '<span class="trace-dot"></span> Link ready.';
      resultLink.textContent = shortUrl;
      resultLink.href = shortUrl;
      resultLink.parentElement.style.transition = 'opacity 0.3s ease';
      resultLink.parentElement.style.opacity = '1';

      if (demoHint) demoHint.textContent = '';
    } catch (err) {
      setError('Network error. Please try again.');
    }

    demoBtn.disabled = false;
    demoBtn.textContent = 'Shorten';
  });

  demoInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') demoBtn.click();
  });

  demoInput.addEventListener('input', () => {
    if (demoHint) demoHint.textContent = 'Free demo: shorten up to 3 links without an account.';
  });

  copyBtn.addEventListener('click', () => {
    const text = resultLink.textContent;
    if (!text) return;

    navigator.clipboard.writeText(text).then(() => {
      const originalText = copyBtn.textContent;
      copyBtn.textContent = 'Copied!';
      copyBtn.style.color = '#27c93f';
      copyBtn.style.borderColor = '#27c93f';

      setTimeout(() => {
        copyBtn.textContent = originalText;
        copyBtn.style.color = '';
        copyBtn.style.borderColor = '';
      }, 2000);
    });
  });

  function setError(message) {
    demoOutput.classList.remove('hidden');
    resultLink.parentElement.style.opacity = '0';
    processingText.innerHTML = `<span class="trace-dot"></span> ${message}`;
    if (demoHint) demoHint.textContent = '';
  }
});
