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

  demoBtn.addEventListener('click', () => {
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
    processingText.innerHTML = '<span class="trace-dot"></span> Tracing route…';

    setTimeout(() => {
      processingText.innerHTML = '<span class="trace-dot"></span> Route traced. Link ready.';

      setTimeout(() => {
        const slug = Math.random().toString(36).substring(2, 8);
        const shortUrl = `short.url/${slug}`;

        resultLink.textContent = shortUrl;
        resultLink.href = `https://${shortUrl}`;

        resultLink.parentElement.style.transition = 'opacity 0.3s ease';
        resultLink.parentElement.style.opacity = '1';

        demoBtn.disabled = false;
        demoBtn.textContent = 'Shorten';
      }, 400);
    }, 600);
  });

  demoInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') demoBtn.click();
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
});
