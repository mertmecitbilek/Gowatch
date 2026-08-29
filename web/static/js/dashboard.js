/**
 * GoWatch Dashboard JavaScript
 * Monitor ekleme, silme, toggle ve modal yönetimi
 */

document.addEventListener('DOMContentLoaded', () => {
    // Modal yönetimi
    const overlay = document.getElementById('modal-overlay');
    const addBtn = document.getElementById('add-monitor-btn');
    const emptyAddBtn = document.getElementById('empty-add-btn');
    const closeBtn = document.getElementById('modal-close');
    const cancelBtn = document.getElementById('cancel-btn');
    const form = document.getElementById('add-monitor-form');
    const typeSelect = document.getElementById('m-type');

    function openModal() {
        overlay.classList.add('active');
        document.getElementById('m-name').focus();
    }

    function closeModal() {
        overlay.classList.remove('active');
        form.reset();
    }

    if (addBtn) addBtn.addEventListener('click', openModal);
    if (emptyAddBtn) emptyAddBtn.addEventListener('click', openModal);
    if (closeBtn) closeBtn.addEventListener('click', closeModal);
    if (cancelBtn) cancelBtn.addEventListener('click', closeModal);
    overlay?.addEventListener('click', (e) => {
        if (e.target === overlay) closeModal();
    });

    // Escape key
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') closeModal();
    });

    // Monitor türüne göre form alanlarını göster/gizle
    if (typeSelect) {
        typeSelect.addEventListener('change', () => {
            const type = typeSelect.value;
            const portGroup = document.getElementById('port-group');
            const httpOptions = document.getElementById('http-options');
            const urlLabel = document.getElementById('url-label');
            const urlInput = document.getElementById('m-url');

            switch (type) {
                case 'http':
                    portGroup.style.display = 'none';
                    httpOptions.style.display = '';
                    urlLabel.textContent = 'URL *';
                    urlInput.placeholder = 'https://example.com';
                    break;
                case 'tcp':
                    portGroup.style.display = '';
                    httpOptions.style.display = 'none';
                    urlLabel.textContent = 'Host *';
                    urlInput.placeholder = 'example.com';
                    break;
                case 'ping':
                    portGroup.style.display = 'none';
                    httpOptions.style.display = 'none';
                    urlLabel.textContent = 'Host *';
                    urlInput.placeholder = 'example.com or 8.8.8.8';
                    break;
                case 'dns':
                    portGroup.style.display = 'none';
                    httpOptions.style.display = 'none';
                    urlLabel.textContent = 'Domain *';
                    urlInput.placeholder = 'example.com';
                    break;
            }
        });
    }

    // Monitor oluşturma formu submit
    if (form) {
        form.addEventListener('submit', async (e) => {
            e.preventDefault();

            const saveBtn = document.getElementById('save-btn');
            saveBtn.disabled = true;
            saveBtn.textContent = 'Saving...';

            
        const notifCheckboxes = document.querySelectorAll('.notif-checkbox:checked');
        const notifIds = Array.from(notifCheckboxes).map(cb => parseInt(cb.value));

        const payload = {

                name: document.getElementById('m-name').value.trim(),
                type: document.getElementById('m-type').value,
                url: document.getElementById('m-url').value.trim(),
                port: parseInt(document.getElementById('m-port').value) || 0,
                interval: parseInt(document.getElementById('m-interval').value) || 60,
                timeout: parseInt(document.getElementById('m-timeout').value) || 30,
                retries: parseInt(document.getElementById('m-retries').value) || 1,
                accepted_codes: document.getElementById('m-codes').value || '200-299',
                description: document.getElementById('m-description').value.trim(),
            notification_ids: JSON.stringify(notifIds),
            };

            try {
                const resp = await fetch('/api/monitors', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });

                const data = await resp.json();

                if (resp.ok) {
                    showToast('Monitor created successfully!', 'success');
                    closeModal();
                    setTimeout(() => location.reload(), 800);
                } else {
                    showToast(data.error || 'Failed to create monitor', 'error');
                }
            } catch (err) {
                showToast('Network error: ' + err.message, 'error');
            } finally {
                saveBtn.disabled = false;
                saveBtn.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg> Save Monitor';
            }
        });
    }
});

// Monitor toggle
async function toggleMonitor(id) {
    try {
        const resp = await fetch(`/api/monitors/${id}/toggle`, { method: 'POST' });
        const data = await resp.json();
        if (resp.ok) {
            showToast(data.data.active ? 'Monitor activated' : 'Monitor paused', 'info');
            setTimeout(() => location.reload(), 600);
        }
    } catch (err) {
        showToast('Error: ' + err.message, 'error');
    }
}

// Monitor sil
async function deleteMonitor(id, name) {
    if (!confirm(`Delete monitor "${name}"? This action cannot be undone.`)) return;

    try {
        const resp = await fetch(`/api/monitors/${id}`, { method: 'DELETE' });
        if (resp.ok) {
            showToast('Monitor deleted', 'success');
            const card = document.getElementById(`monitor-${id}`);
            if (card) {
                card.style.transition = 'opacity 0.3s, transform 0.3s';
                card.style.opacity = '0';
                card.style.transform = 'translateX(-20px)';
                setTimeout(() => { card.remove(); updateStats(); }, 300);
            }
        }
    } catch (err) {
        showToast('Error: ' + err.message, 'error');
    }
}

function updateStats() {
    const cards = document.querySelectorAll('.monitor-card');
    let up = 0, down = 0, pending = 0;
    cards.forEach(card => {
        const s = parseInt(card.dataset.status);
        if (s === 1) up++;
        else if (s === 0) down++;
        else pending++;
    });
    const set = (id, v) => { const e = document.getElementById(id); if (e) e.textContent = v; };
    set('stat-up', up);
    set('stat-down', down);
    set('stat-pending', pending);
    set('stat-total', cards.length);
}
