/**
 * ShortURL — Landing Page Interactive Elements
 */

document.addEventListener('DOMContentLoaded', () => {
  const demoInput = document.getElementById('demo-input');
  const demoBtn = document.getElementById('demo-btn');
  const demoOutput = document.getElementById('demo-output');
  const resultLink = document.getElementById('result-link');
  const copyBtn = document.getElementById('copy-btn');
  const processingText = document.querySelector('.processing');

  // Interactive Terminal Demo
  demoBtn.addEventListener('click', () => {
    const url = demoInput.value.trim();
    
    if (!url) {
      demoInput.style.border = '1px solid #ff5f56';
      setTimeout(() => demoInput.style.border = 'none', 1000);
      return;
    }

    // Reset state
    demoBtn.disabled = true;
    demoBtn.textContent = '...';
    demoOutput.classList.remove('hidden');
    resultLink.parentElement.style.opacity = '0';
    processingText.innerHTML = '[analyzing payload]...';
    
    // Simulate API delay
    setTimeout(() => {
      processingText.innerHTML = '[processing telemetry]... <span class="success">OK</span>';
      
      setTimeout(() => {
        const slug = Math.random().toString(36).substring(2, 8);
        const shortUrl = `short.url/${slug}`;
        
        resultLink.textContent = shortUrl;
        resultLink.href = `https://${shortUrl}`;
        
        resultLink.parentElement.style.transition = 'opacity 0.3s ease';
        resultLink.parentElement.style.opacity = '1';
        
        demoBtn.disabled = false;
        demoBtn.textContent = 'Execute';
      }, 400);
      
    }, 600);
  });

  // Enter key support
  demoInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
      demoBtn.click();
    }
  });

  // Copy functionality
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
